package main

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/registry/proto"
)

const PORT = 10000
const SERVER = "server"
const BALANCER = "balancer"
const PROTO = "proto"
const HANDLER = "handler"
const CHUNK_SIZE = 1024

type registryServer struct {
	Protos   map[string][]byte
	Handlers map[string][]byte
}

func (s *registryServer) Push(stream pb.Registry_PushServer) error {
	data := make([]byte, 0)
	filetype := ""
	name := ""
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			switch filetype {
			case PROTO:
				s.Handlers[name] = data
			case HANDLER:
				s.Protos[name] = data
			default:
				grpclog.Fatal("Empty push request")
			}

			return stream.SendAndClose(&pb.Received{
				Received: true,
			})
		}

		filetype = chunk.FileType
		name = chunk.Name
		check(err)
		data = append(data, chunk.Data...)
	}

	return nil
}

func (s *registryServer) Pull(req *pb.Request, stream pb.Registry_PullServer) error {
	var data []byte
	switch req.FileType {
	case PROTO:
		data = s.Protos[req.Name]
	case HANDLER:
		data = s.Handlers[req.Name]
	}

	r := bytes.NewReader(data)
	chunk := make([]byte, CHUNK_SIZE)
	for {
		_, err := r.Read(chunk)
		if err == io.EOF {
			return nil
		}
		err = stream.Send(&pb.Chunk{
			FileType: req.FileType,
			Name:     req.Name,
			Data:     chunk,
		})
		check(err)
	}

	return nil
}

func initServer() *registryServer {
	s := new(registryServer)
	s.Protos = make(map[string][]byte)
	s.Handlers = make(map[string][]byte)

	return s
}

func check(err error) {
	if err != nil {
		grpclog.Fatal(err)
	}
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	check(err)

	grpcServer := grpc.NewServer()
	pb.RegisterRegistryServer(grpcServer, initServer())
	grpcServer.Serve(lis)
}
