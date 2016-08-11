package codegen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/protoc-gen-go/generator"
)

func Generate(file []byte, name string) ([]byte, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	protofile := filepath.Join(dir, fmt.Sprintf("%s.proto", name))
	fmt.Println(protofile)
	n := bytes.IndexByte(file, 0)
	err = ioutil.WriteFile(protofile, file[:n], 0644)
	if err != nil {
		return nil, err
	}
	defer os.Remove(protofile)

	go_out := fmt.Sprintf("--go_out=%s", dir)
	proto_path := fmt.Sprintf("--proto_path=%s", dir)
	cmd := exec.Command("protoc", go_out, proto_path, protofile)
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	defer os.Remove(fmt.Sprintf("%s/%s.pb.go", dir, name))

	pbfile := fmt.Sprintf("%s.pb.go", name)
	pb, err := ioutil.ReadFile(pbfile)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
