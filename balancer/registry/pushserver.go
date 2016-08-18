package lbreg

import (
	r "github.com/open-lambda/code-registry/registry"
	"github.com/open-lambda/load-balancer/balancer/inspect/codegen"
)

type LBFileProcessor struct{}

func (p LBFileProcessor) Process(name string, files map[string][]byte) ([]r.DBInsert, error) {
	ret := make([]r.DBInsert, 0)
	pb, err := codegen.GenPB(files[PROTO], name)
	if err != nil {
		return ret, err
	}

	parser, err := codegen.GenParser(name, files[PROTO])
	if err != nil {
		return ret, err
	}

	sfiles := map[string]interface{}{
		"id":      name,
		"handler": files[HANDLER],
		"pb":      pb,
	}
	sinsert := r.DBInsert{
		Table: SERVER,
		Data:  &sfiles,
	}
	ret = append(ret, sinsert)

	lbfiles := map[string]interface{}{
		"id":     name,
		"parser": parser,
	}
	lbinsert := r.DBInsert{
		Table: BALANCER,
		Data:  &lbfiles,
	}
	ret = append(ret, lbinsert)

	return ret, nil
}

func InitPushServer(cluster []string) *r.PushServer {
	proc := LBFileProcessor{}
	return r.InitPushServer(cluster, DATABASE, proc, SPORT, CHUNK_SIZE)
}
