package registry

import (
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/open-lambda/load-balancer/balancer/registry/regproto"
)

func (c *PushClient) sendFile(stream pb.Registry_PushClient, name, filetype string, data []byte) {
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
		grpcCheck(err)
	}

	return
}

func (c *PushClient) Push(name, proto, handler string) {
	stream, err := c.Conn.Push(context.Background())
	grpcCheck(err)

	data, err := ioutil.ReadFile(proto)
	grpcCheck(err)
	c.sendFile(stream, name, PROTO, data)

	data, err = ioutil.ReadFile(handler)
	grpcCheck(err)
	c.sendFile(stream, name, HANDLER, data)

	_, err = stream.CloseAndRecv()
	grpcCheck(err)

	return
}

func InitPushClient(serveraddr string, chunksize int) *PushClient {
	c := new(PushClient)

	c.ServerAddr = serveraddr
	c.ChunkSize = chunksize

	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(c.ServerAddr, opts...)
	grpcCheck(err)
	defer conn.Close()

	c.Conn = pb.NewRegistryClient(conn)

	return c
}
