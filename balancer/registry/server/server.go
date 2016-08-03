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
	protos   map[string][]byte
	handlers map[string][]byte
}

func (s *registryServer) Push(stream pb.Registry_PushServer) error {
	for {
		// chunk, err
		_, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Received{
				Received: true,
			})
		}
		// actually read the chunks of data, if it fails send false
	}

	return nil
}

func (s *registryServer) Pull(req *pb.Request, stream pb.Registry_PullServer) error {
	var data []byte
	switch req.FileType {
	case PROTO:
		data = s.protos[req.Name]
	case HANDLER:
		data = s.handlers[req.Name]
	}

	r := bytes.NewReader(data)
	chunk := make([]byte, CHUNK_SIZE)
	for {
		_, err := r.Read(chunk)
		err2 := stream.Send(&pb.Chunk{
			FileType: req.FileType,
			Name:     req.Name,
			Data:     chunk,
		})
		if err2 != nil {
			return err2
		}

		if err == io.EOF {
			return nil
		}
	}

	return nil
}

func initServer() *registryServer {
	s := new(registryServer)
	s.protos = make(map[string][]byte)
	s.handlers = make(map[string][]byte)

	return s
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRegistryServer(grpcServer, initServer())
	grpcServer.Serve(lis)
}
