package lbreg

import r "github.com/open-lambda/code-registry/registry"

func InitLBPullClient(cluster []string) *LBPullClient {
	c := LBPullClient{
		Client: r.InitPullClient(cluster, DATABASE, BALANCER),
	}

	return &c
}

func (c *LBPullClient) Pull(name string) LBFiles {
	files := c.Client.Pull(name)

	ret := LBFiles{
		Parser: files[PARSER].([]byte),
	}

	return ret
}

func InitServerPullClient(cluster []string) *ServerPullClient {
	c := ServerPullClient{
		Client: r.InitPullClient(cluster, DATABASE, SERVER),
	}

	return &c
}

func (c *ServerPullClient) Pull(name string) ServerFiles {
	files := c.Client.Pull(name)

	ret := ServerFiles{
		Handler: files[HANDLER].([]byte),
		PB:      files[PB].([]byte),
	}

	return ret
}
