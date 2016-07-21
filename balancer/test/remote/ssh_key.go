package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type Reponse struct {
	Key struct {
		ID          int
		Fingerprint string
		PublicKey   string
		Name        string
	}
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

func main() {
	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Fatal("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	if len(os.Args) != 3 {
		log.Fatal("Usage: ./ssh_key /path/to/public/key key_name")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	pubkey, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	request := &godo.KeyCreateRequest{
		Name:      os.Args[2],
		PublicKey: string(pubkey),
	}

	response, _, err := client.Keys.Create(request)
	if err != nil {
		log.Fatal(fmt.Sprintf("Key creation failed with: %v", err))
	}

	fmt.Printf("Creation of the key '%s' was successful\n", response.Name)
	fmt.Println("Please add the following line to your ~/.bashrc file to use the remote test:")
	fmt.Printf("export DO_PUBLIC_KEY_ID=%d\n", response.ID)
}
