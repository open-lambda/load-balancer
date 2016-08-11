package registry

import (
	"bytes"
	"io/ioutil"
	"log"

	r "gopkg.in/dancannon/gorethink.v2"
)

func bytesToString(b []byte) string {
	n := bytes.IndexByte(b, 0)

	return string(b[:n])
}

func (c *PullClient) Pull(name string) map[string][]byte {
	ret := make(map[string][]byte)

	switch c.Type {
	case SERVER:
		res, err := r.Table(SERVER).Get(name).Run(c.Conn)
		check(err)

		files := ServerFiles{}
		res.One(&files)
		check(res.Err())

		ret["handler"] = files.Handler
		ret["pb"] = files.PB

		return ret

	case BALANCER:
		res, err := r.Table(BALANCER).Get(name).Run(c.Conn)
		check(err)

		files := BalancerFiles{}
		res.One(&files)
		check(res.Err())

		ret["parser"] = files.Parser

		return ret
	}

	return ret
}

func writeStringToFile(s, filename string) {
	raw := []byte(s)
	err := ioutil.WriteFile(filename, raw, 0644)
	check(err)

	return
}

func initClient(cluster []string, db string, clienttype string) *PullClient {
	c := new(PullClient)
	c.Type = clienttype

	session, err := r.Connect(r.ConnectOpts{
		Addresses: cluster,
		Database:  db,
	})
	check(err)

	c.Conn = session

	return c
}

func InitLBClient(cluster []string, db string) *PullClient {
	return initClient(cluster, db, BALANCER)
}

func InitServerClient(cluster []string, db string) *PullClient {
	return initClient(cluster, db, SERVER)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
