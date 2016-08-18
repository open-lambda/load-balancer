package main

import (
	"fmt"
	r "github.com/open-lambda/load-balancer/balancer/registry"
)

func main() {
	cluster := []string{"127.0.0.1:28015"}
	pushs := r.InitPushServer(cluster)
	fmt.Println("Running pushserver...")
	pushs.Run()
}
