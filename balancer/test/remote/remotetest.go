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

const CLIENT_ITERATIONS = 10000

const BASE_IMAGE = "ubuntu-14-04-x64"
const NUM_CLIENTS = 1

const SERVER_BINARY = "runserver"
const BALANCER_BINARY = "runbalancer"
const CLIENT_BINARY = "runclient"

const BALANCER_CONF = "balancer.conf"
const CLIENT_CONF = "client.conf"
const SSH_CONF = "ssh.conf"

const BALANCER_PORT = "50051"
const SERVER_PORT = "8080"

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

func WriteClientConfig(filename string, balancer_ip string) {
	conf := ClientConfig{
		LBAddr:     fmt.Sprintf("%s:%s", balancer_ip, BALANCER_PORT),
		Iterations: CLIENT_ITERATIONS,
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
	cmd := exec.Command("ssh", "-F", sshconf, fmt.Sprintf("root@%s", ip), fmt.Sprintf("\"sh -c 'nohup ./%s > /dev/null 2>&1 &'\"", name))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	fmt.Printf("%s\n", cmd.Path)
	fmt.Printf("%v\n", cmd.Args)
	err := cmd.Run()
	check(err)
}

func SCP(name string, ip string, dir string) {
	sshconf := filepath.Join(dir, SSH_CONF)
	cmd := exec.Command("scp", "-F", sshconf, filepath.Join(dir, name), fmt.Sprintf("root@%s:./%s", ip, name))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	fmt.Printf("%s\n", cmd.Path)
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

// IF SSH IS FAILING FOR SOME REASON CHANGE PERMISSIONS ON YOUR PRIVATE KEY TO 600
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

	client_wg.Wait()

	fmt.Println("Waiting for SSH to come up...")
	time.Sleep(30 * time.Second)

	// scp runserver binary for servers and run them
	fmt.Println("Copying and running runserver binaries...")
	for k := range server_ips {
		server_wg.Add(1)
		go func() {
			defer server_wg.Done()
			SCP(SERVER_BINARY, server_ips[k], dir)
			EXEC(SERVER_BINARY, server_ips[k], dir)
		}()
	}

	fmt.Println("Copying runbalancer binaries and config files...")
	SCP(BALANCER_CONF, balancer_ip, dir)
	SCP(BALANCER_BINARY, balancer_ip, dir)

	server_wg.Wait()

	fmt.Println("Running loadbalancer...")
	EXEC(BALANCER_BINARY, balancer_ip, dir)

	// scp client config files
	fmt.Println("Copying client binaries and config files...")
	for k := range client_ips {
		client_wg.Add(1)
		go func() {
			defer client_wg.Done()
			SCP(CLIENT_CONF, client_ips[k], dir)
			SCP(CLIENT_BINARY, client_ips[k], dir)
		}()
	}

	client_wg.Wait()

	// run clients
	// TODO add timeout?
	fmt.Println("Running clients...")
	for k := range client_ips {
		test_wg.Add(1)
		go func() {
			defer test_wg.Done()
			EXEC(CLIENT_BINARY, client_ips[k], dir)
		}()
	}

	test_wg.Wait()
}
