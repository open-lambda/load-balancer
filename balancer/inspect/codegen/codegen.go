package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func genPB(dir, protofile, name string) ([]byte, error) {
	go_out := fmt.Sprintf("--go_out=%s", dir)
	proto_path := fmt.Sprintf("--proto_path=%s", dir)

	cmd := exec.Command("protoc", go_out, proto_path, protofile)
	var e bytes.Buffer
	cmd.Stderr = &e

	err := cmd.Run()
	if err != nil {
		return nil, errors.New(e.String())
	}

	pbfile := fmt.Sprintf("%s/%s.pb.go", dir, name)
	defer os.Remove(pbfile)

	pb, err := ioutil.ReadFile(pbfile)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func writeProto(file []byte, dir, name string) (string, error) {
	protofile := filepath.Join(dir, fmt.Sprintf("%s.proto", name))
	n := bytes.IndexByte(file, 0)

	err := ioutil.WriteFile(protofile, file[:n], 0644)
	if err != nil {
		return "", err
	}

	return protofile, nil
}

func Generate(file []byte, name string) ([]byte, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	protofile, err := writeProto(file, dir, name)
	if err != nil {
		return nil, err
	}
	defer os.Remove(protofile)

	return genPB(dir, protofile, name)
}
