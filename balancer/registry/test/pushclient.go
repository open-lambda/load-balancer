package main

import (
	"fmt"
	r "github.com/open-lambda/load-balancer/balancer/registry"
)

const (
	SERVER_ADDR = "127.0.0.1:10000"
	SERVER_PORT = 10000
	CHUNK_SIZE  = 1024

	NAME         = "TEST"
	PROTO_PUSH   = "proto.in"
	PROTO_PULL   = "proto.out"
	HANDLER_PUSH = "handler.in"
	HANDLER_PULL = "handler.out"
)

func main() {
	pushc := r.InitPushClient(SERVER_ADDR, CHUNK_SIZE)
	fmt.Println("Pushing from client...")
	pushc.Push(NAME, PROTO_PUSH, HANDLER_PUSH)
}
