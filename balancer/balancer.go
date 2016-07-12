package main

import (
    "bytes"
    "container/list"
	"fmt"
    "io"
	"log"
	"net"

    "github.com/open-lambda/load-balancer/balancer/connPeek"
    "github.com/open-lambda/load-balancer/balancer/serverPick"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/transport"

	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	lbAddr    = "localhost:50051" // balancer address
)

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
			panic("oh no!")
		}

		// Will need access to buf later for proxying
        var buf bytes.Buffer
        r := io.TeeReader(conn1, &buf)

        // Using conn to peek at the method name w/o affecting buf
        var conn net.Conn = &connPeek.ReaderConn{Reader: r, Conn: conn1}
        st, err := transport.NewServerTransport("http2", conn, 100, nil)
        if err != nil {
            panic(err.Error())
        }

        st.HandleStreams(func(stream *transport.Stream) {
            // Get method name
            name := stream.Method()

            // Make decision about which backend to connect to
            servers, err := chooser.ChooseServers(name, *list.New())
            fmt.Printf("Server chosen to run on: %v\n", servers[0])
            conn2, err := net.Dial("tcp", servers[0])
            if err != nil {
                panic(err.Error())
            }

            // Proxy between client & chosen server
            go io.Copy(conn1, conn2)
            go io.Copy(conn2, &buf)
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
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name /*, Num: 5*/})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}

func main() {
    servers := []string{"localhost:5052", "localhost:5053", "localhost:5054"}
    for i := 0; i < len(servers); i++ {
        go runServer(servers[i])
    }
    chooser := serverPick.NewRandPicker(servers)
	go runBalancer(lbAddr, chooser)
	runClient(lbAddr)
}
