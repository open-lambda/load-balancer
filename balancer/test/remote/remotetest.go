package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const BASE_IMAGE = "ubuntu-14-04-x64"

const SERVER_BINARY = "runserver"
const BALANCER_BINARY = "runbalancer"
const CLIENT_BINARY = "runclient"

const BALANCER_CONF = "balancer.conf"
const CLIENT_CONF = "client.conf"
const SSH_CONF = "ssh.conf"
const TEST_OUTPUT = "test.out"

const BALANCER_PORT = "50051"
const SERVER_PORT = "50052"

type ClientOutput struct {
	Avglatency float64
	Name       string
	Region     string
	Size       string
	Latencies  []int64
}

type TestOutput struct {
	Avglatency float64
	Clients    []ClientOutput
}

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

// Required for oauth2 authentication
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}

	return token, nil
}

// Writes the compiled test outputs from all of the clients as JSON to TEST_OUTPUT
func WriteTestOutput(filename string, latencies [][]int64, avglatencies []float64, clients []godo.Droplet) {
	clientoutputs := make([]ClientOutput, len(avglatencies))
	avglatency := 0.0
	for k := range avglatencies {
		clientoutputs[k] = ClientOutput{
			Latencies:  latencies[k],
			Avglatency: avglatencies[k],
			Region:     clients[k].Region.Slug,
			Size:       clients[k].Size.Slug,
			Name:       clients[k].Name,
		}
		avglatency += avglatencies[k]
	}

	aggregateavg := avglatency / float64(len(avglatencies))
	fmt.Printf("Average latency across all clients: \n%fns\n", aggregateavg)

	testoutput := TestOutput{
		Avglatency: aggregateavg,
		Clients:    clientoutputs,
	}

	json, err := json.MarshalIndent(testoutput, "", "    ")
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// Writes the configuration file for the client droplets (specified by ClientConfig struct)
func WriteClientConfig(filename string, balancer_ip string, iterations int) {
	conf := ClientConfig{
		LBAddr:     fmt.Sprintf("%s:%s", balancer_ip, BALANCER_PORT),
		Iterations: iterations,
	}

	json, err := json.MarshalIndent(conf, "", "    ")
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// Writes the configuration file for the balancer droplet (specified by LBConfig struct)
func WriteLBConfig(filename string, servers []string) {
	formatted := make([]string, len(servers))
	for k := range servers {
		formatted[k] = fmt.Sprintf("%s:%s", servers[k], SERVER_PORT)
	}
	conf := LBConfig{
		Servers: formatted,
		LBPort:  BALANCER_PORT,
	}

	json, err := json.MarshalIndent(conf, "", "    ")
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// Parses the output returned by executing the "runclient" binaries as a way for them
// to return the test latencies from their perspective.
func ParseClientOutput(out string) ([]int64, float64) {
	latencies := make([]int64, 0)
	var avglatency float64

	lines := strings.Split(out, "\n")
	for k := range lines {
		split := strings.Split(lines[k], ":")
		if len(split) != 2 {
			log.Panic("Client output incorrectly formatted")
		}

		switch split[0] {
		case "latency":
			latency, err := strconv.ParseInt(split[1], 10, 64)
			check(err)

			latencies = append(latencies, latency)

		case "avglatency":
			avglatency, err := strconv.ParseFloat(split[1], 64)
			check(err)

			return latencies, avglatency
		}

	}

	log.Panic("Client output incorrectly formatted. Expected avglatency")
	return latencies, avglatency
}

// Executes a binary on a remote machine. Uses a custom ssh config file to suppress warnings. Returns stdout output
func EXEC(binary string, ip string, dir string) string {
	sshconf := filepath.Join(dir, SSH_CONF)
	cmd := exec.Command("ssh", "-F", sshconf, fmt.Sprintf("root@%s", ip), fmt.Sprintf("./%s", binary))
	fmt.Printf("%v\n", cmd.Args)

	out, err := cmd.Output()
	check(err)

	return string(out)
}

// Copies a file to a remote machine. Uses a custom ssh config file to suppress warnings. Returns stdout output
func SCP(file string, ip string, dir string) string {
	sshconf := filepath.Join(dir, SSH_CONF)
	cmd := exec.Command("scp", "-F", sshconf, filepath.Join(dir, file), fmt.Sprintf("root@%s:./%s", ip, file))
	fmt.Printf("%v\n", cmd.Args)

	out, err := cmd.Output()
	check(err)

	return string(out)
}

// Waits for the droplet with the given 'id' to have 'active' status and a public ip address
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

// Creates and returns droplets specified by a list of DropletConfig structs
func DropletsFromConfig(client *godo.Client, keys []godo.DropletCreateSSHKey, conf []DropletConfig, name string) []godo.Droplet {
	droplets := make([]godo.Droplet, 0)
	for k := range conf {
		names := make([]string, conf[k].Number)
		for i := 0; i < conf[k].Number; i++ {
			names[i] = fmt.Sprintf("lbtest-%s-%d-%d", name, k, i)
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

		newdroplets, _, err := client.Droplets.CreateMultiple(request)
		check(err)
		droplets = append(droplets, newdroplets...)

	}

	return droplets
}

// Deletes all droplets linked with the user's account whose names start with "lbtest-"
func DeleteTestDroplets(client *godo.Client) {
	droplets := GetAllDroplets(client)

	for k := range droplets {
		if !strings.HasPrefix(droplets[k].Name, "lbtest-") {
			continue
		}
		_, err := client.Droplets.Delete(droplets[k].ID)

		if err != nil {
			log.Printf("Deletion of droplet %s failed. Please manually destroy it.", droplets[k].Name)
		}
	}
	return
}

// Returns all droplets linked with the user's account
func GetAllDroplets(client *godo.Client) []godo.Droplet {
	options := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	droplets, _, err := client.Droplets.List(options)
	check(err)

	return droplets
}

// Reads the test configuration file and returns a TestConfig struct
func ReadTestConfig(filename string) TestConfig {
	fd, err := os.Open(filename)
	check(err)

	decoder := json.NewDecoder(fd)
	conf := TestConfig{}

	err = decoder.Decode(&conf)
	check(err)

	return conf
}

// Returns all SSH public keys linked with the user's account
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

// Authenticates with the DigitalOcean personal access token and returns the godo client object
func GetClient() *godo.Client {
	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Panic("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	return godo.NewClient(oauthClient)
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}

	return
}

func main() {
	// get directory containing the binary
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	check(err)

	client := GetClient()
	defer DeleteTestDroplets(client)

	// gets all available ssh keys from the digitalocean account (all are added to created droplets)
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

	server_ips := make([]string, len(servers))
	for k := range servers {
		// start goroutines to wait for servers
		server_wg.Add(1)
		go func(id int, idx int) {
			defer server_wg.Done()
			server_ips[idx] = WaitForDroplet(client, id)
		}(servers[k].ID, k)
	}

	// spin up droplet for load balancer
	fmt.Println("Initializing loadbalancer...")
	balancers := DropletsFromConfig(client, keys, conf.Balancers, "loadbalancer")
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
		<----- WRITE CONFIG FILES ----->
	*/

	// write config for the load balancer
	server_wg.Wait()
	fmt.Println("Writing loadbalancer configuration...")
	WriteLBConfig(filepath.Join(dir, BALANCER_CONF), server_ips)

	// write config for clients
	balancer_wg.Wait()
	fmt.Println("Writing client configuration...")
	WriteClientConfig(filepath.Join(dir, CLIENT_CONF), balancer_ip, conf.Iterations)

	// give the droplets some time for SSH to initialize
	client_wg.Wait()
	fmt.Println("Waiting for SSH to come up...")
	time.Sleep(30 * time.Second)

	/*
		<----- COPY FILES ----->
	*/

	// scp runserver binary for servers
	fmt.Println("Copying runserver binaries...")
	for k := range server_ips {
		server_wg.Add(1)
		go func(ip string) {
			defer server_wg.Done()
			SCP(SERVER_BINARY, ip, dir)
		}(server_ips[k])
	}

	// scp runbalancer and config files for balancer
	fmt.Println("Copying runbalancer binaries and config files...")
	SCP(BALANCER_CONF, balancer_ip, dir)
	SCP(BALANCER_BINARY, balancer_ip, dir)

	// scp runclient and client config files
	fmt.Println("Copying client binaries and config files...")
	for k := range client_ips {
		client_wg.Add(1)
		go func(ip string) {
			defer client_wg.Done()
			SCP(CLIENT_CONF, ip, dir)
			SCP(CLIENT_BINARY, ip, dir)
		}(client_ips[k])
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
	}

	// waits for the servers to be run - should find a better way to do this
	time.Sleep(3 * time.Second)

	// run loadbalancer
	fmt.Println("Running loadbalancer...")
	go EXEC(BALANCER_BINARY, balancer_ip, dir)

	// waits for the loadbalancer to be run - should find a better way to do this
	time.Sleep(3 * time.Second)

	// run clients
	fmt.Println("Running clients...")
	latencies := make([][]int64, len(clients))
	avglatencies := make([]float64, len(clients))
	for k := range client_ips {
		test_wg.Add(1)
		go func(ip string, idx int) {
			defer test_wg.Done()
			latencies[idx], avglatencies[idx] = ParseClientOutput(EXEC(CLIENT_BINARY, ip, dir))
		}(client_ips[k], k)
	}

	test_wg.Wait()

	// write client outputs
	fmt.Println("Writing test output...")
	WriteTestOutput(filepath.Join(dir, TEST_OUTPUT), latencies, avglatencies, clients)

	fmt.Printf("Testing complete. Results written to %s\n", TEST_OUTPUT)
}
