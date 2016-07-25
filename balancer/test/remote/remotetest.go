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

const BALANCER_CONF = "balancer.conf"
const CLIENT_CONF = "client.conf"
const SSH_CONF = "ssh.conf"

const BALANCER_PORT = "50051"
const SERVER_PORT = "50052"

type DropletConfig struct {
	Region string
	Size   string
	Number int
}

type TestConfig struct {
	Servers    []DropletConfig
	Clients    []DropletConfig
	Balancers  []DropletConfig
	Iterations int
}

type LBConfig struct {
	Servers []string
	LBPort  string
}

type ClientConfig struct {
	LBAddr     string
	Iterations int
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

func WriteClientConfig(filename string, balancer_ip string, iterations int) {
	conf := ClientConfig{
		LBAddr:     fmt.Sprintf("%s:%s", balancer_ip, BALANCER_PORT),
		Iterations: iterations,
	}

	json, err := json.Marshal(conf)
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

func WriteLBConfig(filename string, servers []string) {
	formatted := make([]string, len(servers))
	for k := range servers {
		formatted[k] = fmt.Sprintf("%s:%s", servers[k], SERVER_PORT)
	}
	conf := LBConfig{
		Servers: formatted,
		LBPort:  BALANCER_PORT,
	}

	json, err := json.Marshal(conf)
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

func EXEC(name string, ip string, dir string) {
	sshconf := filepath.Join(dir, SSH_CONF)
	//cmd := exec.Command("ssh", "-n", "-F", sshconf, fmt.Sprintf("root@%s", ip), fmt.Sprintf("\"sh -c 'nohup ./%s > /dev/null 2>&1 &'\"", name))
	cmd := exec.Command("ssh", "-F", sshconf, fmt.Sprintf("root@%s", ip), fmt.Sprintf("./%s", name))
	//cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	fmt.Printf("%v\n", cmd.Args)
	err := cmd.Run()
	check(err)
}

func SCP(name string, ip string, dir string) {
	sshconf := filepath.Join(dir, SSH_CONF)
	cmd := exec.Command("scp", "-F", sshconf, filepath.Join(dir, name), fmt.Sprintf("root@%s:./%s", ip, name))
	//cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	fmt.Printf("%v\n", cmd.Args)
	err := cmd.Run()
	check(err)
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
		newservers := CreateDroplets(client, request)
		servers = append(servers, newservers...)

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

	/*
		<----- INITIALIZE DROPLETS ----->
	*/

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
		go func(id int, idx int) {
			defer server_wg.Done()
			server_ips[idx] = WaitForDroplet(client, id)
		}(servers[k].ID, k)
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
		go func(id int, idx int) {
			defer client_wg.Done()
			client_ips[idx] = WaitForDroplet(client, id)
		}(clients[k].ID, k)
	}

	/*
		<----- WRITE FILES ----->
	*/

	// write config for the load balancer
	server_wg.Wait()
	fmt.Println("Writing loadbalancer configuration...")
	WriteLBConfig(filepath.Join(dir, BALANCER_CONF), server_ips)

	// write config for clients
	balancer_wg.Wait()
	fmt.Println("Writing client configuration...")
	WriteClientConfig(filepath.Join(dir, CLIENT_CONF), balancer_ip, conf.Iterations)

	fmt.Println("Waiting for SSH to come up...")
	time.Sleep(30 * time.Second)
	client_wg.Wait() // don't need to touch the client ips until we scp

	/*
		<----- COPY FILES ----->
	*/

	// scp runserver binary for servers
	fmt.Println("Copying runserver binaries...")
	for k := range server_ips {
		server_wg.Add(1)
		go func(idx int) {
			defer server_wg.Done()
			SCP(SERVER_BINARY, server_ips[idx], dir)
		}(k)
	}

	// scp runbalancer and config files for balancer
	fmt.Println("Copying runbalancer binaries and config files...")
	SCP(BALANCER_CONF, balancer_ip, dir)
	SCP(BALANCER_BINARY, balancer_ip, dir)

	// scp runclient and client config files
	fmt.Println("Copying client binaries and config files...")
	for k := range client_ips {
		client_wg.Add(1)
		go func(idx int) {
			defer client_wg.Done()
			SCP(CLIENT_CONF, client_ips[idx], dir)
			SCP(CLIENT_BINARY, client_ips[idx], dir)
		}(k)
	}

	server_wg.Wait()
	client_wg.Wait()

	/*
		<----- RUN BINARIES ----->
	*/

	// run servers
	fmt.Println("Running servers...")
	for k := range server_ips {
		go EXEC(SERVER_BINARY, server_ips[k], dir)
		/*
			server_wg.Add(1)
			go func(idx int) {
				defer server_wg.Done()
				EXEC(SERVER_BINARY, server_ips[idx], dir)
			}(k)
		*/
	}

	//server_wg.Wait()
	time.Sleep(5 * time.Second) // TODO fix this

	// run loadbalancer
	fmt.Println("Running loadbalancer...")
	go EXEC(BALANCER_BINARY, balancer_ip, dir)

	time.Sleep(5 * time.Second) // TODO fix this

	// run clients
	// TODO add timeout?
	fmt.Println("Running clients...")
	for k := range client_ips {
		test_wg.Add(1)
		go func(idx int) {
			defer test_wg.Done()
			EXEC(CLIENT_BINARY, client_ips[idx], dir)
		}(k)
	}

	test_wg.Wait()
}
