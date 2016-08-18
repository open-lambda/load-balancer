package main

import (
	"fmt"
	r "github.com/open-lambda/load-balancer/balancer/registry"
)

func main() {
	saddr := fmt.Sprintf("127.0.0.1:%d", r.SPORT)
	pushc := r.InitPushClient(saddr)
	fmt.Println("Pushing from client...")
	files := r.PushClientFiles{
		Proto:   "proto.in",
		Handler: "handler.in",
	}
	pushc.PushFiles("test", files)
}
