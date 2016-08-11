package main

import (
	"fmt"
	r "github.com/open-lambda/load-balancer/balancer/registry"
)

const (
	SERVER_ADDR = "127.0.0.1:10000"
	SERVER_PORT = 10000
	CHUNK_SIZE  = 1024
	DATABASE    = "registry"

	NAME         = "TEST"
	PROTO_PUSH   = "proto.in"
	PROTO_PULL   = "proto.out"
	HANDLER_PUSH = "handler.in"
	HANDLER_PULL = "handler.out"
)

func main() {
	CLUSTER := []string{"127.0.0.1:28015"}
	pushs := r.InitPushServer(CLUSTER, DATABASE, SERVER_PORT, CHUNK_SIZE)
	fmt.Println("Running pushserver...")
	pushs.Run()
}
