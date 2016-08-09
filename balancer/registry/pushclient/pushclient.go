// TODO: send both of the files in push
package pushclient

import (
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/open-lambda/load-balancer/balancer/dbregistry/proto"
)

const PROTO = "proto"
const HANDLER = "handler"

type PushClient struct {
	ServerAddr string
	ChunkSize  int
	Conn       pb.RegistryClient
}

func (c *PushClient) SendFile(stream pb.Registry_PushClient, name, filetype string, data []byte) {
	r := bytes.NewReader(data)
	for {
		chunk := make([]byte, c.ChunkSize)
		n, err := r.Read(chunk)
		if n == 0 && err == io.EOF {
			return
		}

		err = stream.Send(&pb.Chunk{
			Name:     name,
			Data:     chunk,
			FileType: filetype,
		})
		check(err)
	}

	return
}

func (c *PushClient) Push(name, proto, handler string) {
	stream, err := c.Conn.Push(context.Background())
	check(err)

	data, err := ioutil.ReadFile(proto)
	check(err)
	c.SendFile(stream, name, PROTO, data)

	data, err = ioutil.ReadFile(handler)
	check(err)
	c.SendFile(stream, name, HANDLER, data)

	_, err = stream.CloseAndRecv()
	check(err)

	return
}

func Init(serveraddr string, chunksize int) *PushClient {
	c := new(PushClient)

	c.ServerAddr = serveraddr
	c.ChunkSize = chunksize

	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(c.ServerAddr, opts...)
	check(err)
	defer conn.Close()

	c.Conn = pb.NewRegistryClient(conn)

	return c
}

func check(err error) {
	if err != nil {
		grpclog.Fatal(err)
	}
}
