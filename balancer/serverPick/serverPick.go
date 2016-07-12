package serverPick

import (
	"container/list"
	"math/rand"
	"time"
)

type ServerPicker interface {
	ChooseServers(name string, params list.List) (servers []string, err error)
	RegisterTimes(servers []string, times []float64)
}

type RandPicker struct {
	rg      rand.Rand
	servers []string
}

func NewRandPicker(servers []string) RandPicker {
	src := rand.NewSource(time.Now().UnixNano())
	return *&RandPicker{servers: servers, rg: *rand.New(src)}

}

func (rp RandPicker) ChooseServers(name string, params list.List) (servers []string, err error) {
	index := rp.rg.Intn(len(rp.servers))
	return []string{rp.servers[index]}, nil
}

func (rp RandPicker) RegisterTimes(servers []string, times []float64) {
	return
}
