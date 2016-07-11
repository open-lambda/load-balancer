package main

import (
    "bytes"
	"fmt"
    "io"
	"log"
	"net"

    "github.com/open-lambda/load-balancer/balancer/connPeek"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/transport"

	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	port1     = ":50051" // balancer
	port2     = ":50052" // server
	innerPort = ":50053" // port for intermediate connection to listen to
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	fmt.Printf("Inside SayHello (server)\n")
	return &pb.HelloReply{Message: "Hi " + in.Name}, nil
}

func runServer() {
	lis, err := net.Listen("tcp", port2)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	s.Serve(lis)
}

func runBalancer() {
	lis, err := net.Listen("tcp", "localhost" + port1)
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
            fmt.Printf("method from inside lb: %v\n", name)

            // Make decision about which backend to connect to
            conn2, err := net.Dial("tcp", "localhost" + port2)
            if err != nil {
                panic(err.Error())
            }

            // Proxy between client & chosen server
            go io.Copy(conn1, conn2)
            go io.Copy(conn2, &buf)
        })

    }
}

func runClient() {
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost"+port1, grpc.WithInsecure())
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
	go runServer()
	go runBalancer()
	runClient()
}
