package main

import "github.com/open-lambda/load-balancer/balancer/test/server"

func main() {
	server.RunServer(":50052")
}
