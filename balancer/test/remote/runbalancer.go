package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/open-lambda/load-balancer/balancer"
	"github.com/open-lambda/load-balancer/balancer/serverPick"
)

type Config struct {
	Servers []string
	LBPort  string
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
	conf := readConfig("balancer.conf")
	chooser := serverPick.NewFirstTwo(conf.Servers)

	balancer.RunBalancer(fmt.Sprintf(":%s", conf.LBPort), chooser)
}
