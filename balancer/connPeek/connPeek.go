package connPeek

import (
	"io"
	"net"
	"time"
)

/*
 * Main purpose of this struct is a hack to enable connection peeking without
 * disrupting the ability of the loadbalancer to proxy between client & server.
 */
type ReaderConn struct {
	Reader io.Reader
	Conn   net.Conn
	conn2  net.Conn
}

/*
 * Read and write will operate on the reader so that we can still read from the
 * connection!
 */
func (r *ReaderConn) Read(b []byte) (n int, err error) {
	return r.Reader.Read(b)
}

func (r *ReaderConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

/*
 * This function is a little strange, but it's a way to setup a clean way to
 * close the connection on the server end.
 */
func (r *ReaderConn) SecondConn(c2 net.Conn) {
	r.conn2 = c2
}

/*
 * This function gets called automatically by grpc's HandleStreams. This way we
 * also know it's safe to close the connection to the server at the same time.
 */
func (r *ReaderConn) Close() error {
	r.conn2.Close()
	return r.Conn.Close()
}

/*
 * All the rest just call the functions from Conn
 */
func (r *ReaderConn) LocalAddr() net.Addr {
	return r.Conn.LocalAddr()
}

func (r *ReaderConn) RemoteAddr() net.Addr {
	return r.Conn.RemoteAddr()
}

func (r *ReaderConn) SetDeadline(t time.Time) error {
	return r.Conn.SetDeadline(t)
}

func (r *ReaderConn) SetReadDeadline(t time.Time) error {
	return r.Conn.SetReadDeadline(t)
}

func (r *ReaderConn) SetWriteDeadline(t time.Time) error {
	return r.Conn.SetWriteDeadline(t)
}
