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

// Always picks first two servers
type FirstTwo struct {
	servers []string
}

func NewFirstTwo(servers []string) FirstTwo {
	return *&FirstTwo{servers: servers}
}

func (ft FirstTwo) ChooseServers(name string, params list.List) (servers []string, err error) {
	return []string{ft.servers[0], ft.servers[1]}, nil
}

func (ft FirstTwo) RegisterTimes(servers []string, times []float64) {
	return
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

func (rp RandPicker) ChooseServers(name string, params list.List) (servers []string, err error) {
	index := rp.rg.Intn(len(rp.servers))
	return []string{rp.servers[index]}, nil
}

func (rp RandPicker) RegisterTimes(servers []string, times []float64) {
	return
}
