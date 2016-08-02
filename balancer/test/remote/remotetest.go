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
const TEST_CONF = "test.conf"
const TEST_OUTPUT = "test.out"

const BALANCER_PORT = "50051"
const SERVER_PORT = "50052"

const LB_CONSUMERS = 10

type Tester struct {
	Client  *godo.Client
	Conf    TestConfig
	Clients []godo.Droplet
	Servers []godo.Droplet
	LB      godo.Droplet
	Keys    []godo.DropletCreateSSHKey
	Dir     string
}

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
	Servers   []string
	LBPort    string
	Consumers int
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
func (t *Tester) WriteTestOutput(latencies [][]int64, avglatencies []float64) {
	filename := filepath.Join(t.Dir, TEST_OUTPUT)

	clientoutputs := make([]ClientOutput, len(avglatencies))
	avglatency := 0.0
	for k := range avglatencies {
		clientoutputs[k] = ClientOutput{
			Latencies:  latencies[k],
			Avglatency: avglatencies[k],
			Region:     t.Clients[k].Region.Slug,
			Size:       t.Clients[k].Size.Slug,
			Name:       t.Clients[k].Name,
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
func (t *Tester) WriteClientConfig(balancer_ip string) {
	filename := filepath.Join(t.Dir, CLIENT_CONF)

	conf := ClientConfig{
		LBAddr:     fmt.Sprintf("%s:%s", balancer_ip, BALANCER_PORT),
		Iterations: t.Conf.Iterations,
	}

	json, err := json.MarshalIndent(conf, "", "    ")
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// Writes the configuration file for the balancer droplet (specified by LBConfig struct)
func (t *Tester) WriteLBConfig(servers []string) {
	filename := filepath.Join(t.Dir, BALANCER_CONF)

	formatted := make([]string, len(servers))
	for k := range servers {
		formatted[k] = fmt.Sprintf("%s:%s", servers[k], SERVER_PORT)
	}

	conf := LBConfig{
		Servers:   formatted,
		LBPort:    BALANCER_PORT,
		Consumers: LB_CONSUMERS,
	}

	json, err := json.MarshalIndent(conf, "", "    ")
	check(err)

	err = ioutil.WriteFile(filename, json, 0644)
	check(err)

	return
}

// Parses the output returned by executing the "runclient" binaries as a way for them
// to return the test latencies from their perspective.
func (t *Tester) ParseClientOutput(out string) ([]int64, float64) {
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
func (t *Tester) EXEC(binary string, ip string) string {
	sshconf := filepath.Join(t.Dir, SSH_CONF)
	cmd := exec.Command("ssh", "-F", sshconf, fmt.Sprintf("root@%s", ip), fmt.Sprintf("./%s", binary))
	fmt.Printf("%v\n", cmd.Args)

	out, err := cmd.Output()
	check(err)

	return string(out)
}

// Copies a file to a remote machine. Uses a custom ssh config file to suppress warnings. Returns stdout output
func (t *Tester) SCP(file string, ip string) string {
	sshconf := filepath.Join(t.Dir, SSH_CONF)
	cmd := exec.Command("scp", "-F", sshconf, filepath.Join(t.Dir, file), fmt.Sprintf("root@%s:./%s", ip, file))
	fmt.Printf("%v\n", cmd.Args)

	out, err := cmd.Output()
	check(err)

	return string(out)
}

// Waits for the droplet with the given 'id' to have 'active' status and a public ip address
func (t *Tester) WaitForDroplet(id int) string {
	for {
		droplet, _, err := t.Client.Droplets.Get(id)
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
func (t *Tester) DropletsFromConfig(confs []DropletConfig, name string) []godo.Droplet {
	droplets := make([]godo.Droplet, 0)
	for k := range confs {
		conf := confs[k]
		names := make([]string, conf.Number)
		for i := 0; i < conf.Number; i++ {
			names[i] = fmt.Sprintf("lbtest-%s-%d-%d", name, k, i)
		}
		request := &godo.DropletMultiCreateRequest{
			Names:  names,
			Region: conf.Region,
			Size:   conf.Size,
			Image: godo.DropletCreateImage{
				Slug: BASE_IMAGE,
			},
			SSHKeys: t.Keys,
		}

		newdroplets, _, err := t.Client.Droplets.CreateMultiple(request)
		check(err)
		droplets = append(droplets, newdroplets...)

	}

	return droplets
}

func (t *Tester) DeleteDroplet(droplet godo.Droplet) {
	_, err := t.Client.Droplets.Delete(droplet.ID)

	if err != nil {
		log.Printf("Deletion of droplet %s failed. Please manually destroy it.", droplet.Name)
	}

	return
}

// Deletes all droplets linked with the user's account whose names start with "lbtest-"
func (t *Tester) DeleteDroplets() {
	for k := range t.Servers {
		t.DeleteDroplet(t.Servers[k])
	}
	for k := range t.Clients {
		t.DeleteDroplet(t.Clients[k])
	}
	t.DeleteDroplet(t.LB)

	return
}

// Reads the test configuration file and returns a TestConfig struct
func (t *Tester) ReadTestConfig() {
	fd, err := os.Open(filepath.Join(t.Dir, TEST_CONF))
	check(err)

	decoder := json.NewDecoder(fd)
	t.Conf = TestConfig{}

	err = decoder.Decode(&t.Conf)
	check(err)

	return
}

// Returns all SSH public keys linked with the user's account
func (t *Tester) GetKeys() {
	options := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	keys, _, err := t.Client.Keys.List(options)
	check(err)

	key_requests := make([]godo.DropletCreateSSHKey, len(keys))
	for k := range keys {
		key_requests[k] = godo.DropletCreateSSHKey{
			ID:          keys[k].ID,
			Fingerprint: keys[k].Fingerprint,
		}
	}

	t.Keys = key_requests

	return
}

// Authenticates with the DigitalOcean personal access token and returns the godo client object
func (t *Tester) GetClient() {
	pat := os.Getenv("DO_AUTHENTICATION_TOKEN")
	if pat == "" {
		log.Panic("DO_AUTHENTICATION_TOKEN environment variable not set")
	}

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	t.Client = godo.NewClient(oauthClient)

	return
}

func (t *Tester) GetDir() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	check(err)
	t.Dir = dir

	return
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}

	return
}

func main() {
	t := new(Tester)
	defer t.DeleteDroplets()

	t.GetDir()

	t.GetClient()

	// gets all available ssh keys from the digitalocean account (all are added to created droplets)
	t.GetKeys()

	t.ReadTestConfig()

	var server_wg sync.WaitGroup
	var balancer_wg sync.WaitGroup
	var client_wg sync.WaitGroup
	var test_wg sync.WaitGroup

	/*
		<----- INITIALIZE DROPLETS ----->
	*/

	// spin up droplets for servers
	fmt.Println("Initializing servers...")
	t.Servers = t.DropletsFromConfig(t.Conf.Servers, "server")

	server_ips := make([]string, len(t.Servers))
	for k := range t.Servers {
		// start goroutines to wait for servers
		server_wg.Add(1)
		go func(id int, idx int) {
			defer server_wg.Done()
			server_ips[idx] = t.WaitForDroplet(id)
		}(t.Servers[k].ID, k)
	}

	// spin up droplet for load balancer
	fmt.Println("Initializing loadbalancer...")
	t.LB = t.DropletsFromConfig(t.Conf.Balancers, "loadbalancer")[0]

	// start goroutine to wait for balancer
	var balancer_ip string
	balancer_wg.Add(1)
	go func() {
		defer balancer_wg.Done()
		balancer_ip = t.WaitForDroplet(t.LB.ID)
	}()

	// spin up droplets for clients
	fmt.Println("Initializing clients...")
	t.Clients = t.DropletsFromConfig(t.Conf.Clients, "client")

	// start goroutines to wait for clients
	client_ips := make([]string, len(t.Clients))
	for k := range t.Clients {
		client_wg.Add(1)
		go func(id int, idx int) {
			defer client_wg.Done()
			client_ips[idx] = t.WaitForDroplet(id)
		}(t.Clients[k].ID, k)
	}

	/*
		<----- WRITE CONFIG FILES ----->
	*/

	// write config for the load balancer
	server_wg.Wait()
	fmt.Println("Writing loadbalancer configuration...")
	t.WriteLBConfig(server_ips)

	// write config for clients
	balancer_wg.Wait()
	fmt.Println("Writing client configuration...")
	t.WriteClientConfig(balancer_ip)

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
			t.SCP(SERVER_BINARY, ip)
		}(server_ips[k])
	}

	// scp runbalancer and config files for balancer
	fmt.Println("Copying runbalancer binaries and config files...")
	t.SCP(BALANCER_CONF, balancer_ip)
	t.SCP(BALANCER_BINARY, balancer_ip)

	// scp runclient and client config files
	fmt.Println("Copying client binaries and config files...")
	for k := range client_ips {
		client_wg.Add(1)
		go func(ip string) {
			defer client_wg.Done()
			t.SCP(CLIENT_CONF, ip)
			t.SCP(CLIENT_BINARY, ip)
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
		go t.EXEC(SERVER_BINARY, server_ips[k])
	}

	// waits for the servers to be run - should find a better way to do this
	time.Sleep(3 * time.Second)

	// run loadbalancer
	fmt.Println("Running loadbalancer...")
	go t.EXEC(BALANCER_BINARY, balancer_ip)

	// waits for the loadbalancer to be run - should find a better way to do this
	time.Sleep(3 * time.Second)

	// run clients
	fmt.Println("Running clients...")
	latencies := make([][]int64, len(t.Clients))
	avglatencies := make([]float64, len(t.Clients))
	for k := range client_ips {
		test_wg.Add(1)
		go func(ip string, idx int) {
			defer test_wg.Done()
			latencies[idx], avglatencies[idx] = t.ParseClientOutput(t.EXEC(CLIENT_BINARY, ip))
		}(client_ips[k], k)
	}

	test_wg.Wait()

	// write client outputs
	fmt.Println("Writing test output...")
	t.WriteTestOutput(latencies, avglatencies)

	fmt.Printf("Testing complete. Results written to %s\n", TEST_OUTPUT)
}
