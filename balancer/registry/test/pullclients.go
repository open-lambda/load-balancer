package main

import (
	"fmt"
	"io/ioutil"
	"log"

	r "github.com/open-lambda/load-balancer/balancer/registry"
)

const (
	SERVER_ADDR = "127.0.0.1:10000"
	SERVER_PORT = 10000
	CHUNK_SIZE  = 1024

	NAME         = "TEST"
	PROTO_PULL   = "proto.out"
	HANDLER_PULL = "handler.out"
	PARSER_PULL  = "so.out"
	DATABASE     = "registry"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	CLUSTER := []string{"127.0.0.1:28015"}
	spull := r.InitServerClient(CLUSTER, DATABASE)
	fmt.Println("Running pullclient as a server...")
	sfiles := spull.Pull(NAME)
	handler := sfiles["handler"]
	pb := sfiles["pb"]

	lbpull := r.InitLBClient(CLUSTER, DATABASE)
	fmt.Println("Running pullclient as a balancer...")
	lbfiles := lbpull.Pull(NAME)
	parser := lbfiles["parser"]

	err := ioutil.WriteFile(PROTO_PULL, pb, 0644)
	check(err)
	err = ioutil.WriteFile(HANDLER_PULL, handler, 0644)
	check(err)
	err = ioutil.WriteFile(PARSER_PULL, parser, 0644)
	check(err)
}
