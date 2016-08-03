package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

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
		err2 := stream.Send(&pb.Chunk{
			FileType: filetype,
			Name:     name,
			Data:     chunk,
		})

		check(err2)

		if err == io.EOF {
			return
		}
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
		data = append(data, chunk.Data...)
		if err == io.EOF {
			return data
		}
		check(err)
	}

	return data
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
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

	// Push proto
	push(client, name, PROTO, protofile)

	// Push handler
	push(client, name, HANDLER, handlerfile)

	// Pull proto
	fmt.Printf(string(pull(client, name, PROTO, "proto.out")))

	// Pull handler
	fmt.Printf(string(pull(client, name, HANDLER, "handler.out")))
}
