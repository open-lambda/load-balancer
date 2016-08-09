package main

import (
	"fmt"

	"github.com/open-lambda/load-balancer/balancer/dbregistry/pullclient"
	"github.com/open-lambda/load-balancer/balancer/dbregistry/pushclient"
	"github.com/open-lambda/load-balancer/balancer/dbregistry/pushserver"
)

const (
	SERVER   = "server"
	BALANCER = "balancer"

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
	CLUSTER := []string{"127.0.0.1:28015"}
	pushs := pushserver.Init(CLUSTER, SERVER_PORT, CHUNK_SIZE)
	pushc := pushclient.Init(SERVER_ADDR, CHUNK_SIZE)
	spull := pullclient.Init(CLUSTER, SERVER)
	lbpull := pullclient.Init(CLUSTER, BALANCER)

	fmt.Println("Running pushserver...")
	go pushs.Run()

	fmt.Println("Pushing from client...")
	pushc.Push(NAME, PROTO_PUSH, HANDLER_PUSH)

	fmt.Println("Running pullclient as a server...")
	spull.Pull(NAME)

	fmt.Println("Running pullclient as a balancer...")
	lbpull.Pull(NAME)
}
