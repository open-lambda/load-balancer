package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const BASE_IMAGE = "ubuntu-14-04-x64"
const NUM_CLIENTS = 1

type DropletConfig struct {
	Region string
	Size   string
	Number int
}

type TestConfig struct {
	Servers   []DropletConfig
	Clients   []DropletConfig
	Balancers []DropletConfig
}

type LBConfig struct {
	Servers []string
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}

	return token, nil
}

func DropletsFromConfig(client *godo.Client, conf []DropletConfig, name string) []godo.Droplet {
	servers := make([]godo.Droplet, 0)
	for k := 0; k < len(conf.Servers); k++ {
		if conf.Servers[k].Number == 1 {
			request := &godo.DropletCreateRequest{
				Name:   fmt.Sprintf("%s-%d", name, k),
				Region: conf.Servers[k].Region,
				Size:   conf.Servers[k].Size,
				Image: godo.DropletCreateImage{
					Slug: BASE_IMAGE,
				},
			}
			servers = append(servers, []godo.Droplet{*CreateDroplet(client, request)}...)

		} else {
			names := make([]string, conf.Servers[k].Number)
			for i := 0; i < conf.Servers[k].Number; i++ {
				names[i] = fmt.Sprintf("%s-%d-%d", name, k, i)
			}

			request := &godo.DropletMultiCreateRequest{
				Names:  names,
				Region: conf.Servers[k].Region,
				Size:   conf.Servers[k].Size,
				Image: godo.DropletCreateImage{
					Slug: BASE_IMAGE,
				},
			}
			servers = append(servers, CreateDroplets(client, request)...)
		}

	}

	return servers
}

func CreateDroplets(client *godo.Client, request *godo.DropletMultiCreateRequest) []godo.Droplet {
	newDroplets, _, err := client.Droplets.CreateMultiple(request)

	if err != nil {
		log.Fatal(err)
	}

	return newDroplets
}

func CreateDroplet(client *godo.Client, request *godo.DropletCreateRequest) *godo.Droplet {
	newDroplet, _, err := client.Droplets.Create(request)

	if err != nil {
		log.Fatal(err)
	}

	return newDroplet
}

func ReadTestConfig(filename string) *TestConfig {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(fd)
	conf := TestConfig{}

	err = decoder.Decode(&conf)
	if err != nil {
		log.Fatalf("could not decode config file: %v", err)
	}

	return &conf
}

func main() {
	if os.Getenv("DO_PUBLIC_KEY_ID") == "" {
		log.Fatal("DO_PUBLIC_KEY_ID environment variable not set. Please use ssh_key tool before attempting remote test again")
	}

	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Fatal("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	conf := ReadServerConfig("test.conf")

	// spin up droplets for servers
	servers := DropletsFromConfig(client, conf.Servers, "server")

	// TODO scp runserver binary for servers
	// TODO write config for the load balancer

	// TODO spin up droplet for load balancer
	balancers := DropletsFromConfig(client, conf.Balancers, "loadbalancer")
	// TODO write config for the clients

	// spin up droplets for clients
	clients := DropletsFromConfig(client, conf.Clients, "client")
	// TODO scp runclient binary and client.conf config file

	// TODO run clients (with a timeout)

	// TODO delete droplets
}
