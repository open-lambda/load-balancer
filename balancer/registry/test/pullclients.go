package main

import (
	"fmt"
	"io/ioutil"
	"log"

	r "github.com/open-lambda/load-balancer/balancer/registry"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	cluster := []string{"127.0.0.1:28015"}

	spull := r.InitServerPullClient(cluster)
	fmt.Println("Running pullclient as a server...")
	sfiles := spull.Pull("test")
	handler := sfiles.Handler
	pb := sfiles.PB

	lbpull := r.InitLBPullClient(cluster)
	fmt.Println("Running pullclient as a balancer...")
	lbfiles := lbpull.Pull("test")
	parser := lbfiles.Parser

	err := ioutil.WriteFile("test.pb.go", pb, 0644)
	check(err)
	err = ioutil.WriteFile("handler.go", handler, 0644)
	check(err)
	err = ioutil.WriteFile("parser", parser, 0644)
	check(err)
}
