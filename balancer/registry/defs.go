package registry

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/registry/regproto"
	r "gopkg.in/dancannon/gorethink.v2"
)

const (
	PROTO    = "proto"
	HANDLER  = "handler"
	SERVER   = "server"
	BALANCER = "balancer"
)

type PushClient struct {
	ServerAddr string
	ChunkSize  int
	Conn       pb.RegistryClient
}

type PushServer struct {
	Port      int
	ChunkSize int
	Conn      *r.Session // sessions are thread safe?
}

type PullClient struct {
	Type string
	Conn *r.Session
}

type ServerFiles struct {
	Name    string `gorethink:"id,omitempty"`
	Handler []byte `gorethink:"handler"`
	PB      []byte `gorethink:"pb"`
}

type BalancerFiles struct {
	Name   string `gorethink:"id,omitempty"`
	Parser []byte `gorethink:"parser"`
}

func grpcCheck(err error) {
	if err != nil {
		grpclog.Fatal(grpc.ErrorDesc(err))
	}
}
