package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"

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

func runInnerServer(buf io.Reader, inConn net.Conn) {
	lis, err := net.Listen("tcp", innerPort)
	if err != nil {
		panic(err.Error())
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			panic("oh no!")
		}

		st, err := transport.NewServerTransport("http2", conn, 100, nil)
		st.HandleStreams(func(stream *transport.Stream) {
			// Get method name
			name := stream.Method()
			fmt.Printf("method from inside lb: %v\n", name)

			// TODO: Will need to choose a backend here based on name

			// Dial up chosen server & copy contents over
			outConn, err := net.Dial("tcp", "localhost"+port2)

			if err != nil {
				panic("oh no!")
			}

			// Copy things through back to client
			go io.Copy(outConn, buf)
			go io.Copy(inConn, outConn)
		})

	}
}

func runBalancer() {
	lis, err := net.Listen("tcp", port1)
	if err != nil {
		panic(err.Error())
	}

	for {
		conn1, err := lis.Accept()
		if err != nil {
			panic("oh no!")
		}

		// Need a multiwriter to both read input & pass it along?
		var buf1 bytes.Buffer
		//w := io.Writer(&buf1)
		io.Copy(&buf1, conn1)
        fmt.Println("Made it here!")
        var r1, r2 = bytes.NewReader(buf1.Bytes()), bytes.NewReader(buf1.Bytes())

		// Start intermediate server to peek, choose and forward
		go runInnerServer(r1, conn1)

		// Connect to inner server
		innerConn, err := net.Dial("tcp", "localhost"+innerPort)

		// Yeah, this is hacky, likely a cleaner way to do is w/ channels
		for ; err != nil; innerConn, err = net.Dial("tcp", "localhost"+innerPort) {
			fmt.Println("Waiting for innerServer")
		}
		// Forward things onto the inner server
		go io.Copy(innerConn, r2)
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
