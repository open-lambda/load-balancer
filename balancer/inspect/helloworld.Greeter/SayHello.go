package test

import (
    "fmt"
    proto "github.com/golang/protobuf/proto"
    pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func GetArgs(buf []byte) {
    in := &pb.HelloRequest{}

    for i := 0; i < len(buf); i++ {
        err := proto.Unmarshal(buf[i:], in)
        if err == nil {
            fmt.Println(i)
            break
        }
    }
    fmt.Printf("Argument in lb: %v\n", in)
}
