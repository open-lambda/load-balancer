package main

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"os"
	"log"
	"net"
	"encoding/json"

	"github.com/open-lambda/load-balancer/balancer/connPeek"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/transport"

	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	lbAddr = "localhost:50051" // balancer address
)

type Config struct {
	Servers []string
}

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("Inside SayHello (server)\n")
	return &pb.HelloReply{Message: "Hi " + in.Name}, nil
}

func runServer(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	s.Serve(lis)
}

func runBalancer(address string, chooser serverPick.ServerPicker) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		panic(err.Error())
	}

	for {
		conn1, err := lis.Accept()
		if err != nil {
			panic(err.Error())
		}

		// Will need access to buf later for proxying
		var buf bytes.Buffer
		r := io.TeeReader(conn1, &buf)

		// Using conn to peek at the method name w/o affecting buf
		conn := &connPeek.ReaderConn{Reader: r, Conn: conn1}
		st, err := transport.NewServerTransport("http2", conn, 100, nil)
		if err != nil {
			panic(err.Error())
		}

		st.HandleStreams(func(stream *transport.Stream) {
			// Get method name
			name := stream.Method()

			// Make decision about which backend(s) to connect to
			servers, err := chooser.ChooseServers(name, *list.New())
			if err != nil {
				panic(err.Error())
			}

			// Actually send the request to "best" backend (first one for now)
			go func() {
				conn2, err := net.Dial("tcp", servers[0])
				if err != nil {
					panic(err.Error())
				}
				conn.SecondConn(conn2) // This is so we can close conn2 cleanly
				io.Copy(conn2, &buf)
				io.Copy(conn1, conn2)
			}()
		})
	}
}

func runClient(address string) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := "tyler"
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}

func readConfig (filename string) (*Config) {
	fd, err := os.Open("conf.json")
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(fd)
	conf := Config{}

	err = decoder.Decode(&conf)
	if err != nil {
		log.Fatalf("could not decode config file: %v", err)
	}

	return &conf
}

func main() {
	conf := readConfig("conf.json")
	for i := 0; i < len(conf.Servers); i++ {
		go runServer(conf.Servers[i])
	}

	chooser := serverPick.NewFirstTwo(conf.Servers)
	go runBalancer(lbAddr, chooser)
	for i := 0; ; i++ {
		fmt.Printf("Client's been run %v time(s)\n", i)
		runClient(lbAddr)
	}
}
