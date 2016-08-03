package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/registry/proto"
)

const SERVERADDR = "127.0.0.1:10000"

const PROTO = "proto"
const HANDLER = "handler"

const CHUNK_SIZE = 1024

func push(client pb.RegistryClient, name string, filetype string, file string) {
	data, err := ioutil.ReadFile(file)
	check(err)

	stream, err := client.Push(context.Background())

	r := bytes.NewReader(data)
	chunk := make([]byte, CHUNK_SIZE)
	for {
		_, err := r.Read(chunk)
		if err == io.EOF {
			// check that reply was successful
			_, err := stream.CloseAndRecv()
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

func pull(client pb.RegistryClient, name string, filetype string, outfile string) []byte {
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
			return data
		}
		check(err)

		data = append(data, chunk.Data...)
	}

	return data
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

	name := "example"
	protofile := "example.proto"
	handlerfile := "example.handler"

	fmt.Println("Pushing proto...")
	// Push proto
	push(client, name, PROTO, protofile)

	fmt.Println("Pushing handler...")
	// Push handler
	push(client, name, HANDLER, handlerfile)

	fmt.Println("Pulling proto...")
	// Pull proto
	proto := pull(client, name, PROTO, "proto.out")
	err = ioutil.WriteFile("proto.out", proto, 0644)
	check(err)

	fmt.Println("Pulling handler...")
	// Pull handler
	handler := pull(client, name, HANDLER, "handler.out")
	err = ioutil.WriteFile("handler.out", handler, 0644)
	check(err)
}
