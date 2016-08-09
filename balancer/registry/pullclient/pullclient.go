package pullclient

import (
	"io/ioutil"
	"log"

	r "gopkg.in/dancannon/gorethink.v2"
)

const (
	SERVER   = "server"
	BALANCER = "balancer"
	DATABASE = "registry"
)

type ServerFiles struct {
	Name        string `gorethink:"primary_key,omitempty"`
	HandlerFile []byte `gorethink:"handler"`
	PBFile      []byte `gorethink:"pb"`
}

type BalancerFiles struct {
	Name   string `gorethink:"primary_key,omitempty"`
	SOFile []byte `gorethink:"so"`
}

type PullClient struct {
	Type string
	Conn *r.Session
}

func (c *PullClient) Pull(name string) map[string][]byte {
	ret := make(map[string][]byte)

	//id := r.UUID(name)
	switch c.Type {
	case SERVER:
		res, err := r.Table(SERVER).Get(name).Run(c.Conn)
		check(err)

		files := ServerFiles{}
		res.One(&files)
		check(res.Err())

		ret["handler"] = files.HandlerFile
		ret["pb"] = files.PBFile

		return ret
	case BALANCER:
		res, err := r.Table(BALANCER).Get(name).Run(c.Conn)
		check(err)

		files := BalancerFiles{}
		res.One(&files)
		check(res.Err())

		ret["so"] = files.SOFile

		return ret
	}

	return ret
}

func WriteStringToFile(s, filename string) {
	raw := []byte(s)
	err := ioutil.WriteFile(filename, raw, 0644)
	check(err)

	return
}

func Init(cluster []string, clienttype string) *PullClient {
	c := new(PullClient)
	c.Type = clienttype

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  DATABASE,
	})
	check(err)

	c.Conn = session

	return c
}

func InitLBClient(cluster []string) *PullClient {
	return Init(cluster, BALANCER)
}

func InitServerClient(cluster []string) *PullClient {
	return Init(cluster, SERVER)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
