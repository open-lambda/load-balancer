package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/open-lambda/load-balancer/balancer"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
	"github.com/open-lambda/load-balancer/balancer/test/client"
	"github.com/open-lambda/load-balancer/balancer/test/server"
)

type Config struct {
	Servers []string
	LBAddr  string
}

func readConfig(filename string) *Config {
	fd, err := os.Open(filename)
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
	conf := readConfig("local.conf")
	for i := 0; i < len(conf.Servers); i++ {
		go server.RunServer(conf.Servers[i])
	}

	chooser := serverPick.NewRandPicker(conf.Servers)
	lb := new(balancer.LoadBalancer)
	lb.Init(conf.LBAddr, chooser, 5)
	go lb.Run()
	time.Sleep(time.Second)
	for i := 0; ; i++ {
		fmt.Printf("Client's been run %v time(s)\n", i)
		client.RunClient(conf.LBAddr)
	}
}
