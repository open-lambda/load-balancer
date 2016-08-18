package lbreg

import (
	r "github.com/open-lambda/code-registry/registry"
)

const (
	CHUNK_SIZE = 1024
	DATABASE   = "lbregistry"
	BALANCER   = "balancer"
	SERVER     = "server"
	PROTO      = "proto"
	HANDLER    = "handler"
	PB         = "pb"
	PARSER     = "parser"
	SPORT      = 10000
)

type PushClient struct {
	Client *r.PushClient
}

type LBPullClient struct {
	Client *r.PullClient
}

type LBFiles struct {
	Parser []byte
}

type ServerPullClient struct {
	Client *r.PullClient
}

type ServerFiles struct {
	Handler []byte
	PB      []byte
}

type PushClientFiles struct {
	Handler string
	Proto   string
}
