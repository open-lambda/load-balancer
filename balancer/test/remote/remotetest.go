package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const BASE_IMAGE = "ubuntu-14-04-x64"
const NUM_CLIENTS = 1

const SERVER_BINARY = "runserver"
const BALANCER_BINARY = "runbalancer"
const CLIENT_BINARY = "runclient"

const SERVER_BINARY_PATH = "/runserver"
const CLIENT_BINARY_PATH = "/runclient"
const BALANCER_BINARY_PATH = "/runbalancer"

const BALANCER_CONF = "balancer.conf"
const CLIENT_CONF = "client.conf"

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

type ClientConfig struct {
	Balancer string
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

func RunClient(client *godo.Client, droplet godo.Droplet) string {
	_, err := droplet.PublicIPv4()
	check(err)
	return ""
}

// TODO combine these two into a generic
func WriteClientConfig(filename string, balancer string) {
	conf := ClientConfig{Balancer: balancer}

	json, err := json.Marshal(conf)
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

func WriteLBConfig(filename string, servers []string) {
	conf := LBConfig{Servers: servers}

	json, err := json.Marshal(conf)
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// TODO remove redundancy here
func EXEC(localpath string, remotepath string, ip string) {
	cmd := exec.Command("ssh", fmt.Sprintf("root@%s", ip))
	err := cmd.Run()
	check(err)
}

func SCP(localpath string, remotepath string, ip string) {
	cmd := exec.Command("scp", localpath, fmt.Sprintf("root@%s:%s", ip, remotepath))
	err := cmd.Run()
	check(err)
}

func SCPAndExec(localpath string, remotepath string, ip string) {
	cmd := exec.Command("scp", localpath, fmt.Sprintf("root@%s:%s", ip, remotepath))
	err := cmd.Run()
	check(err)

	cmd = exec.Command("ssh", fmt.Sprintf("root@%s", ip), remotepath)
	err = cmd.Run()
	check(err)

	return
}

func WaitForDroplet(client *godo.Client, id int) string {
	for {
		droplet, _, err := client.Droplets.Get(id)
		check(err)

		ip, err := droplet.PublicIPv4()
		check(err)

		if droplet.Status == "active" && ip != "" {
			fmt.Printf("Droplet %s active\n", droplet.Name)
			return ip
		}
		time.Sleep(2 * time.Second)
	}
}

func DropletsFromConfig(client *godo.Client, keys []godo.DropletCreateSSHKey, conf []DropletConfig, name string) []godo.Droplet {
	servers := make([]godo.Droplet, 0)
	for k := range conf {
		if conf[k].Number == 1 {
			request := &godo.DropletCreateRequest{
				Name:   fmt.Sprintf("test-%s-%d", name, k),
				Region: conf[k].Region,
				Size:   conf[k].Size,
				Image: godo.DropletCreateImage{
					Slug: BASE_IMAGE,
				},
				SSHKeys: keys,
			}
			servers = append(servers, []godo.Droplet{*CreateDroplet(client, request)}...)

		} else {
			names := make([]string, conf[k].Number)
			for i := 0; i < conf[k].Number; i++ {
				names[i] = fmt.Sprintf("test-%s-%d-%d", name, k, i)
			}

			request := &godo.DropletMultiCreateRequest{
				Names:  names,
				Region: conf[k].Region,
				Size:   conf[k].Size,
				Image: godo.DropletCreateImage{
					Slug: BASE_IMAGE,
				},
				SSHKeys: keys,
			}
			servers = append(servers, CreateDroplets(client, request)...)
		}

	}

	return servers
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

func CreateDroplets(client *godo.Client, request *godo.DropletMultiCreateRequest) []godo.Droplet {
	newDroplets, _, err := client.Droplets.CreateMultiple(request)
	check(err)

	return newDroplets
}

func CreateDroplet(client *godo.Client, request *godo.DropletCreateRequest) *godo.Droplet {
	newDroplet, _, err := client.Droplets.Create(request)
	check(err)

	return newDroplet
}

func ReadTestConfig(filename string) *TestConfig {
	fd, err := os.Open(filename)
	check(err)

	decoder := json.NewDecoder(fd)
	conf := TestConfig{}

	err = decoder.Decode(&conf)
	check(err)

	return &conf
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

// TODO defer deleting all droplets to make sure they're always deleted
func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	check(err)

	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Fatal("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	keys := GetKeys(client)

	conf := ReadTestConfig(filepath.Join(dir, "test.conf"))

	var server_wg sync.WaitGroup
	var balancer_wg sync.WaitGroup
	var client_wg sync.WaitGroup
	var test_wg sync.WaitGroup

	// spin up droplets for servers
	fmt.Println("Initializing servers...")
	servers := DropletsFromConfig(client, keys, conf.Servers, "server")
	defer DeleteDroplets(client, servers)

	server_ips := make([]string, len(servers))
	for k := range servers {
		ip, err := servers[k].PublicIPv4()
		check(err)
		server_ips[k] = ip

		// start goroutines to wait for servers
		server_wg.Add(1)
		go func() {
			defer server_wg.Done()
			server_ips[k] = WaitForDroplet(client, servers[k].ID)
		}()
	}

	// spin up droplet for load balancer
	// TODO clean this up to not use [0]
	fmt.Println("Initializing loadbalancer...")
	balancers := DropletsFromConfig(client, keys, conf.Balancers, "loadbalancer")
	defer DeleteDroplets(client, balancers)
	balancer := balancers[0]

	// start goroutine to wait for balancer
	var balancer_ip string
	balancer_wg.Add(1)
	go func() {
		defer balancer_wg.Done()
		balancer_ip = WaitForDroplet(client, balancer.ID)
	}()

	// spin up droplets for clients
	fmt.Println("Initializing clients...")
	clients := DropletsFromConfig(client, keys, conf.Clients, "client")
	defer DeleteDroplets(client, clients)

	// start goroutines to wait for clients
	client_ips := make([]string, len(clients))
	for k := range clients {
		client_wg.Add(1)
		go func() {
			defer client_wg.Done()
			client_ips[k] = WaitForDroplet(client, clients[k].ID)
		}()
	}

	// write config for the load balancer
	server_wg.Wait()
	fmt.Println("Writing loadbalancer configuration...")
	WriteLBConfig(filepath.Join(dir, BALANCER_CONF), server_ips)

	// write config for clients
	balancer_wg.Wait()
	fmt.Println("Writing client configuration...")
	WriteClientConfig(filepath.Join(dir, CLIENT_CONF), balancer_ip)

	// TODO wait awhile for ssh to come up
	// scp runserver binary for servers and run them
	for k := range server_ips {
		server_wg.Add(1)
		go func() {
			defer server_wg.Done()
			SCP(filepath.Join(dir, SERVER_BINARY), SERVER_BINARY_PATH, server_ips[k])
			EXEC(filepath.Join(dir, SERVER_BINARY), SERVER_BINARY_PATH, server_ips[k])
		}()
	}

	SCP(filepath.Join(dir, BALANCER_CONF), BALANCER_BINARY_PATH, balancer_ip)
	EXEC(filepath.Join(dir, BALANCER_BINARY), BALANCER_BINARY_PATH, balancer_ip)

	client_wg.Wait()

	// scp client config files
	for k := range client_ips {
		client_wg.Add(1)
		go func() {
			defer client_wg.Done()
			SCP(filepath.Join(dir, CLIENT_CONF), CLIENT_BINARY_PATH, client_ips[k])
		}()
	}

	client_wg.Wait()
	// scp and run client binary
	for k := range client_ips {
		client_wg.Add(1)
		go func() {
			defer client_wg.Done()
			SCP(filepath.Join(dir, CLIENT_CONF), CLIENT_BINARY_PATH, client_ips[k])
		}()
	}

	client_wg.Wait()

	// TODO run clients (with a timeout)

	fmt.Println("Running clients...")
	for k := range clients {
		test_wg.Add(1)
		go func() {
			defer test_wg.Done()
			RunClient(client, clients[k])
		}()
	}

	// TODO delete droplets
	test_wg.Wait()
}
