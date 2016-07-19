package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/open-lambda/load-balancer/balancer"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
	"github.com/open-lambda/load-balancer/balancer/test/client"
	"github.com/open-lambda/load-balancer/balancer/test/server"
)

const (
	lbAddr = "localhost:50051" // balancer address
)

type Config struct {
	Servers []string
}

func readConfig(filename string) *Config {
	fd, err := os.Open("conf.json")
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(fd)
	conf := Config{}

	err = decoder.Decode(&conf)
	if err != nil {
		log.Fatalf("could not decode config file: %v", err)
	}

	return &conf
}

func main() {
	conf := readConfig("conf.json")
	for i := 0; i < len(conf.Servers); i++ {
		go server.RunServer(conf.Servers[i])
	}

	chooser := serverPick.NewFirstTwo(conf.Servers)
	go balancer.RunBalancer(lbAddr, chooser)
	for i := 0; ; i++ {
		fmt.Printf("Client's been run %v time(s)\n", i)
		client.RunClient(lbAddr)
	}
}
