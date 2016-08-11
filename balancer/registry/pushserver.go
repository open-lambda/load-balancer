package registry

import (
	"fmt"
	"io"
	"net"

	"github.com/open-lambda/load-balancer/balancer/inspect/codegen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/registry/regproto"
	r "gopkg.in/dancannon/gorethink.v2"
)

func generateParser(proto []byte) ([]byte, error) {
	return []byte("Fake parser code"), nil
}

func (s *PushServer) ProcessAndStore(name string, proto, handler []byte) error {
	pb, err := codegen.Generate(proto, name)
	grpcCheck(err)

	sfiles := map[string]interface{}{
		"id":      name,
		"handler": handler,
		"pb":      pb,
	}

	parser, err := generateParser(proto)

	lbfiles := map[string]interface{}{
		"id":     name,
		"parser": parser,
	}

	opts := r.InsertOpts{Conflict: "replace"}

	_, err = r.Table(SERVER).Insert(&sfiles, opts).RunWrite(s.Conn)
	grpcCheck(err)

	_, err = r.Table(BALANCER).Insert(&lbfiles, opts).RunWrite(s.Conn)
	grpcCheck(err)

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
		grpcCheck(err)
	}

	return nil
}

func (s *PushServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	grpcCheck(err)

	grpcServer := grpc.NewServer()
	pb.RegisterRegistryServer(grpcServer, s)
	grpcServer.Serve(lis)

	return
}

// TODO add authKey argument to creating the session
// TODO don't create the database every time?
func InitPushServer(cluster []string, db string, port, chunksize int) *PushServer {
	s := new(PushServer)

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  db,
	})
	grpcCheck(err)
	/*
		_, err = r.TableCreate(BALANCER).RunWrite(session)
		grpcCheck(err)

		_, err = r.TableCreate(SERVER).RunWrite(session)
		grpcCheck(err)
	*/
	s.Conn = session
	s.Port = port
	s.ChunkSize = chunksize

	return s
}
