package balancer

import (
	"bytes"
	"container/list"
	"io"
	"net"

	"github.com/open-lambda/load-balancer/balancer/connPeek"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
	"google.golang.org/grpc/transport"
)

func RunBalancer(address string, chooser serverPick.ServerPicker) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		panic(err.Error())
	}

	for {
		conn1, err := lis.Accept()
		if err != nil {
			panic(err.Error())
		}

		// Will need access to buf later for proxying
		var buf bytes.Buffer
		r := io.TeeReader(conn1, &buf)

		// Using conn to peek at the method name w/o affecting buf
		conn := &connPeek.ReaderConn{Reader: r, Conn: conn1}
		st, err := transport.NewServerTransport("http2", conn, 100, nil)
		if err != nil {
			panic(err.Error())
		}

		st.HandleStreams(func(stream *transport.Stream) {
			// Get method name
			name := stream.Method()

			// Make decision about which backend(s) to connect to
			servers, err := chooser.ChooseServers(name, *list.New())
			if err != nil {
				panic(err.Error())
			}

			// Actually send the request to "best" backend (first one for now)
			go func() {
				conn2, err := net.Dial("tcp", servers[0])
				if err != nil {
					panic(err.Error())
				}
				conn.SecondConn(conn2) // This is so we can close conn2 cleanly
				io.Copy(conn2, &buf)
				io.Copy(conn1, conn2)
			}()
		})
	}
}
