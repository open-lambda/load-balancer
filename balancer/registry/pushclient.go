package lbreg

import r "github.com/open-lambda/code-registry/registry"

func InitPushClient(saddr string) *PushClient {
	c := r.InitPushClient(saddr, CHUNK_SIZE)

	return &PushClient{Client: c}
}

func (c *PushClient) PushFiles(name string, files PushClientFiles) {
	proto := r.PushClientFile{Name: files.Proto, Type: PROTO}
	handler := r.PushClientFile{Name: files.Handler, Type: HANDLER}

	c.Client.Push(name, proto, handler)

	return
}
