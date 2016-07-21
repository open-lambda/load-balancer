package main

import (
	"fmt"
	"os"
	"log"
	"encoding/json"

	"github.com/open-lambda/load-balancer/balancer/test/client"
)

type Config struct {
	LBAddr string
	Iterations int
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
	conf := readConfig("client.conf")

	for i := 0; i < conf.Iterations; i++ {
		fmt.Printf("Client's been run %v time(s)\n", i)
		client.RunClient(conf.LBAddr)
	}
}
