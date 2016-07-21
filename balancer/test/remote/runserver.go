package main

import "github.com/open-lambda/load-balancer/balancer/test/server"

func main() {
	go server.RunServer(":8080")
}
