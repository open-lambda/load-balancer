package main

import (
	"log"
	"os"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}

	return token, nil
}

func GetAllDroplets(client *godo.Client) []godo.Droplet {
	options := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	droplets, _, err := client.Droplets.List(options)
	check(err)

	return droplets
}

func DeleteDroplets(client *godo.Client, droplets []godo.Droplet) {
	for k := range droplets {
		_, err := client.Droplets.Delete(droplets[k].ID)
		if err != nil {
			log.Printf("Deletion of droplet %s failed. Please manually destroy it.", droplets[k].Name)
		}
	}
	return
}

func GetKeys(client *godo.Client) []godo.DropletCreateSSHKey {
	options := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	keys, _, err := client.Keys.List(options)
	check(err)

	key_requests := make([]godo.DropletCreateSSHKey, len(keys))
	for k := range keys {
		key_requests[k] = godo.DropletCreateSSHKey{
			ID:          keys[k].ID,
			Fingerprint: keys[k].Fingerprint,
		}
	}

	return key_requests
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}

	return
}

func main() {
	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Fatal("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplets := GetAllDroplets(client)
	DeleteDroplets(client, droplets)
}
