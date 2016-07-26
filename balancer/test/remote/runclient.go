package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/open-lambda/load-balancer/balancer/test/client"
)

type Config struct {
	LBAddr     string
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
	latencies := make([]int64, conf.Iterations)
	var aggregate int64 = 0

	for i := 0; i < conf.Iterations; i++ {
		start := time.Now()
		client.RunClient(conf.LBAddr)
		elapsed := time.Since(start).Nanoseconds()

		fmt.Printf("latency:%d\n", elapsed)
		latencies[i] = elapsed
		aggregate += elapsed
	}
	avglatency := float64(aggregate) / float64(conf.Iterations)
	fmt.Printf("avglatency:%f\n", avglatency)

}
