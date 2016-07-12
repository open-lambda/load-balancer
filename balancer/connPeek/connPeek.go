package connPeek

import (
	"io"
	"net"
	"time"
)

type ReaderConn struct {
	Reader io.Reader
	Conn   net.Conn
}

func (r *ReaderConn) Read(b []byte) (n int, err error) {
	return r.Reader.Read(b)
}

func (r *ReaderConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (r *ReaderConn) Close() error {
	panic("not implemented")
}

func (r *ReaderConn) LocalAddr() net.Addr {
	return r.Conn.LocalAddr()
}

func (r *ReaderConn) RemoteAddr() net.Addr {
	return r.Conn.RemoteAddr()
}

func (r *ReaderConn) SetDeadline(t time.Time) error {
	panic("not implemented")
}

func (r *ReaderConn) SetReadDeadline(t time.Time) error {
	panic("not implemented")
}

func (r *ReaderConn) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
}
