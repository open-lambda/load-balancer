package pushserver

import (
	"fmt"
	"io"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/dbregistry/proto"
	r "gopkg.in/dancannon/gorethink.v2"
)

const (
	DATABASE = "registry"

	PROTO   = "proto"
	HANDLER = "handler"

	BALANCER = "balancer"
	SERVER   = "server"
)

type ServerFiles struct {
	Name        string `gorethink:"primary_key,omitempty"`
	HandlerFile []byte `gorethink:"handler"`
	PBFile      []byte `gorethink:"pb`
}

type BalancerFiles struct {
	Name   string `gorethink:"primary_key,omitempty"`
	SOFile []byte `gorethink:"so"`
}

type PushServer struct {
	Port      int
	ChunkSize int
	Conn      *r.Session // sessions are thread safe?
}

func (s *PushServer) ProcessAndStore(name string, proto, handler []byte) error {
	sfiles := ServerFiles{
		Name:        name,
		HandlerFile: handler,
		PBFile:      proto,
	}

	lbfiles := BalancerFiles{
		Name:   name,
		SOFile: handler,
	}

	_, err := r.Table(SERVER).Replace(sfiles).Run(s.Conn)
	check(err)

	_, err = r.Table(BALANCER).Replace(lbfiles).Run(s.Conn)
	check(err)
	return nil
}

func (s *PushServer) Push(stream pb.Registry_PushServer) error {
	proto := make([]byte, 0)
	handler := make([]byte, 0)
	name := ""

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			if name == "" {
				grpclog.Fatal("Empty push request or name field")
			}

			err = s.ProcessAndStore(name, proto, handler)
			if err != nil {
				return err
			}

			return stream.SendAndClose(&pb.Received{
				Received: true,
			})
		}

		switch chunk.FileType {
		case PROTO:
			proto = append(proto, chunk.Data...)
		case HANDLER:
			handler = append(handler, chunk.Data...)
		}

		name = chunk.Name
		check(err)
	}

	return nil
}

func (s *PushServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	check(err)

	grpcServer := grpc.NewServer()
	pb.RegisterRegistryServer(grpcServer, s)
	grpcServer.Serve(lis)

	return
}

// TODO add authKey argument to creating the session
// TODO don't create the database every time?
func Init(cluster []string, port, chunksize int) *PushServer {
	s := new(PushServer)

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  DATABASE,
	})
	check(err)
	/*
		_, err = r.DBCreate(DATABASE).RunWrite(session)
		check(err)

		opts := r.TableCreateOpts{
			PrimaryKey: "name",
		}

		_, err = r.TableCreate(BALANCER, opts).RunWrite(session)
		check(err)

		_, err = r.TableCreate(SERVER, opts).RunWrite(session)
		check(err)
	*/
	s.Conn = session
	s.Port = port
	s.ChunkSize = chunksize

	return s
}

func check(err error) {
	if err != nil {
		grpclog.Fatal(err)
	}
}
