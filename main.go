package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"./agreement"
	"./blockchain"
	"./config"
	"github.com/korkmazkadir/coordinator/registery"
	"github.com/korkmazkadir/go-rpc-node/node"

	"net/http"
	_ "net/http/pprof"
)

func main() {

	var peerAddresses []string

	peerAddressesFlag := flag.String("peers", "", "semicolon seperated list peer addresses")
	registeryAddressFlag := flag.String("registery", "", "address of the registery service (127.0.0.6:9091)")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		log.Println("data is being piped to stdin")
		peerAddresses = getPeerAddressesFromFile(os.Stdin)
	} else {
		log.Println("stdin is from a terminal")
		peerAddresses = getPeerAddressesFromFlag(peerAddressesFlag)
	}

	//reads configuration from config file
	appConfig := readConfigurationFromFile()

	// inits loggers
	agreementLogger, memoryPoolLogger, blockchainLogger, nodeLogger := initLoggers(appConfig)

	//inits blockchain and memory pool
	BlockPayloadSize := appConfig.Blockchain.BlockPayloadSize
	memoryPool := blockchain.NewMemoryPool(BlockPayloadSize, memoryPoolLogger)
	blockchain := blockchain.NewBlockchain(blockchainLogger)

	pk, sk, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Printf("could not generate key %s", err)
	}

	app := agreement.NewBAStar(appConfig.BAStar, pk, sk, memoryPool, blockchain, agreementLogger)

	gossipNodeBufferSize := appConfig.Network.GossipNodeMessageBufferSize
	gossipNode := node.NewGossipNode(app, gossipNodeBufferSize, nodeLogger)

	hostname := getHostName()

	address, err := gossipNode.Start(hostname)
	if err != nil {
		panic(err)
	}

	pid := os.Getpid()
	log.Printf("PID: %d IPAddress: %s\n", pid, address)

	//sets TC rules
	setTCRules()

	if *registeryAddressFlag != "" {
		peerAddresses = connectRegisteryWaitForPeers(*registeryAddressFlag, address, appConfig.NodeCount, app)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(peerAddresses), func(i, j int) { peerAddresses[i], peerAddresses[j] = peerAddresses[j], peerAddresses[i] })
	peerCount := 0
	for _, peerAddress := range peerAddresses {
		if peerCount == appConfig.Network.PeerCount {
			break
		}
		if address == peerAddress {
			continue
		}
		connectToPeer(peerAddress, gossipNode)
		peerCount++
	}

	app.Start()

	fmt.Println(address)

	go func() {
		log.Println("starting pprof")
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	gossipNode.Wait()

}

func getPublicIPAddress() string {
	publicAddress := os.Getenv("PUBLIC_ADDRESS")
	return publicAddress
}

func connectRegisteryWaitForPeers(registryAddress string, currentNodeAddress string, nodeCount int, app *agreement.BAStar) []string {

	log.Printf("Connecting to the node registery %s\n", registryAddress)

	client := &registery.Client{NetAddress: registryAddress}
	//connects to the registery
	client.Connect()

	publicAddress := getPublicIPAddress()
	if publicAddress != "" {
		log.Printf("Public address is set to %s\n", publicAddress)
		tokens := strings.Split(currentNodeAddress, ":")

		if len(tokens) == 1 {
			currentNodeAddress = publicAddress + ":" + tokens[0]
		} else if len(tokens) == 2 {
			currentNodeAddress = publicAddress + ":" + tokens[1]
		} else {
			panic(fmt.Errorf("Public ip address could not set!!! Length of tokens %d", len(tokens)))
		}

		log.Printf("Current node address is set to %s\n", currentNodeAddress)
	}

	// registers the current node address to the registery
	count := client.AddNode(currentNodeAddress)
	for count != nodeCount {
		log.Printf("Waiting for other nodes %d/%d\n", count, nodeCount)
		count = client.GetNodeCount(currentNodeAddress)
		time.Sleep(1 * time.Second)
	}

	nodeList := client.GetNodeList(currentNodeAddress)
	log.Printf("Node list is ready. Length of it %d\n", len(nodeList))

	//closes registery connection
	client.Close()

	return nodeList
}

func getPeerAddressesFromFile(file *os.File) []string {
	addresses := make([]string, 0)

	inputBytes, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	inputString := string(inputBytes)

	tokens := strings.Split(inputString, "\n")
	for _, token := range tokens {
		if token != "" {
			addresses = append(addresses, token)
		}
	}

	return addresses
}

func getPeerAddressesFromFlag(peerAddresses *string) []string {

	log.Println(*peerAddresses)
	if *peerAddresses != "" {
		addresses := strings.Split(*peerAddresses, ";")
		return addresses
	}

	return nil
}

func connectToPeer(peerAddress string, n *node.GossipNode) {

	peer, err := node.NewRemoteNode(peerAddress)
	if err != nil {
		panic(err)
	}

	err = n.AddPeer(peer)
	if err != nil {
		panic(err)
	}

}

func readConfigurationFromFile() *config.Configuration {
	fileName := "config.json"
	appConfig := config.ReadConfigurationFromFile(fileName)
	appConfig.BAStar.TotalMoney = uint64(appConfig.NodeCount) * (appConfig.BAStar.UserMoney)

	configData, _ := json.MarshalIndent(appConfig, "", "\t")
	log.Printf("Starting node with configuration \n %s \n", string(configData))
	return appConfig
}

func initLoggers(appConfig *config.Configuration) (agreementLogger *log.Logger, memoryPoolLogger *log.Logger, blockchainLogger *log.Logger, nodeLogger *log.Logger) {
	flags := log.Ldate | log.Ltime | log.Lmsgprefix

	if appConfig.Logger.EnableAgreementLoging {
		agreementLogger = log.New(os.Stderr, "[agreement] ", flags)
	} else {
		agreementLogger = log.New(ioutil.Discard, "", 0)
	}

	if appConfig.Logger.EnableMemoryPoolLoging {
		memoryPoolLogger = log.New(os.Stderr, "[memor-pool] ", flags)
	} else {
		memoryPoolLogger = log.New(ioutil.Discard, "", 0)
	}

	if appConfig.Logger.EnableBlockchainLoging {
		blockchainLogger = log.New(os.Stderr, "[blockchain] ", flags)
	} else {
		blockchainLogger = log.New(ioutil.Discard, "", 0)
	}

	if appConfig.Logger.EnableGossipNodeLoging {
		nodeLogger = log.New(os.Stderr, "[node] ", flags)
	} else {
		nodeLogger = log.New(ioutil.Discard, "", 0)
	}

	return
}

func getHostName() string {

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	log.Printf("WARNING: using empty host name rather than %s!!!!!", hostname)

	//return hostname
	return ""
}

func setTCRules() {

	bandwidth := 20
	delay := 100

	log.Printf("Setting TC rules bandwidth %d Mbps Delay %d ms\n", bandwidth, delay)

	tcSetExecutable, err := exec.LookPath("tcset")
	if err != nil {
		log.Println("WARNING: Could not find tcset in path")
		return
	}

	args1 := strings.Fields(fmt.Sprintf("eth0 --rate %dMbps --direction incoming", bandwidth))
	args1 = append([]string{tcSetExecutable}, args1...)

	tcSetIncommingCmd := &exec.Cmd{
		Path:   tcSetExecutable,
		Args:   args1,
		Stderr: os.Stdout,
	}

	err = tcSetIncommingCmd.Run()
	if err != nil {
		panic(fmt.Errorf("tcset incomming error: %s", err))
	}

	args2 := strings.Fields(fmt.Sprintf("eth0 --rate %dMbps --delay %dms --direction outgoing", bandwidth, delay))
	args2 = append([]string{tcSetExecutable}, args2...)

	tcSetOutgoingCmd := &exec.Cmd{
		Path:   tcSetExecutable,
		Args:   args2,
		Stderr: os.Stdout,
	}

	err = tcSetOutgoingCmd.Run()
	if err != nil {
		panic(fmt.Errorf("tcset outgoing error: %s", err))
	}

}
