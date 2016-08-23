package serverPick

import (
	"math/rand"
	"time"
)

type ServerPicker interface {
	ChooseServer(name string, args interface{}) (server string, err error)
	RegisterTime(time float64)
}

// Picks only one server randomly
type RandPicker struct {
	rg      rand.Rand
	servers []string
}

func NewRandPicker(servers []string) RandPicker {
	src := rand.NewSource(time.Now().UnixNano())
	return *&RandPicker{servers: servers, rg: *rand.New(src)}
}

func (rp RandPicker) ChooseServer(name string, args interface{}) (server string, err error) {
	index := rp.rg.Intn(len(rp.servers))
	return rp.servers[index], nil
}

func (rp RandPicker) RegisterTime(time float64) {
	return
}
