package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/registry/proto"
)

const SERVERADDR = "127.0.0.1:10000"

const PROTO = "proto"
const HANDLER = "handler"

const CHUNK_SIZE = 1024

func push(client pb.RegistryClient, name, filetype, file string) {
	stream, err := client.Push(context.Background())
	check(err)

	data, err := ioutil.ReadFile(file)
	check(err)

	r := bytes.NewReader(data)
	for {
		chunk := make([]byte, CHUNK_SIZE)
		n, err := r.Read(chunk)
		if n == 0 && err == io.EOF {
			_, err = stream.CloseAndRecv()
			check(err)
			return
		}

		err = stream.Send(&pb.Chunk{
			FileType: filetype,
			Name:     name,
			Data:     chunk,
		})
		check(err)
	}

	return
}

func pull(client pb.RegistryClient, name, filetype string) string {
	stream, err := client.Pull(context.Background(), &pb.Request{
		ClientType: "balancer",
		FileType:   filetype,
		Name:       name,
	})
	check(err)

	data := make([]byte, 0)
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		check(err)

		data = append(data, chunk.Data...)
	}

	fmt.Print(string(data[:]))
	n := bytes.IndexByte(data, 0)
	s := string(data[:n])
	fmt.Print(s)
	return s
}

func check(err error) {
	if err != nil {
		grpclog.Fatal(err)
	}
}

func main() {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(SERVERADDR, opts...)
	if err != nil {
		grpclog.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewRegistryClient(conn)

	name := "test"
	protofile := "test.proto"
	handlerfile := "test.go"

	fmt.Println("Pushing proto...")
	// Push proto
	push(client, name, PROTO, protofile)

	fmt.Println("Pushing handler...")
	// Push handler
	push(client, name, HANDLER, handlerfile)

	fmt.Println("Pulling proto...")
	// Pull proto
	proto := pull(client, name, PROTO)
	check(err)
	fd, err := os.Create("out.proto")
	check(err)
	fd.WriteString(proto)
	fd.Close()

	fmt.Println("Pulling handler...")
	// Pull handler
	handler := pull(client, name, HANDLER)
	check(err)
	fd, err = os.Create("out.go")
	check(err)
	fd.WriteString(handler)
	fd.Close()
}
