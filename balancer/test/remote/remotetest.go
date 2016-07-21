package main

import (
	"encoding/json"
	"os"
	"log"

	"golang.org/x/oauth2"
	"github.com/digitalocean/godo"
)

type ServerConfig struct {
	Servers []struct {
		Region string
		Size string
		Number int
	}
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token {
		AccessToken: t.AccessToken,
	}

	return token, nil
}

func CreateClients() {
	return
}

func CreateLB() {
	return
}

func CreateServers() {
	return
}

func CreateDroplets(client *godo.Client, request *godo.DropletMultiCreateRequest, num int) []godo.Droplet {
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

func readServerConfig(filename string) *ServerConfig {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(fd)
	conf := ServerConfig{}

	err = decoder.Decode(&conf)
	if err != nil {
		log.Fatalf("could not decode config file: %v", err)
	}

	return &conf
}

func main() {
	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")	
	if pat == "" {
		log.Fatal("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource {
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	// TODO add config file for servers
	// TODO spin up droplets for servers
	// TODO copy runserver binary for servers
	// TODO write config for the load balancer

	// TODO spin up droplet for load balancer
	// TODO write config for the clients

	// TODO spin up droplets for clients
	// TODO copy runclient binary and client.conf config file
	// TODO run clients

	// TODO delete droplets

	dropletName := "super-cool-droplet"
	createRequest := &godo.DropletCreateRequest {
		Name: dropletName,
		Region: "nyc3",
		Size: "512mb",
		Image: godo.DropletCreateImage {
			Slug: "ubuntu-14-04-x64",
		},
	}
	
	//newDroplet, _, err := client.Droplets.Create(createRequest)
	CreateDroplet(client, createRequest)

}
