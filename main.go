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
	"strings"
	"time"

	"./agreement"
	"./blockchain"
	"./config"
	"github.com/korkmazkadir/go-rpc-node/node"
)

const totalStake = 1000
const numberOfNodes = 100

func main() {

	var peerAddresses []string

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		log.Println("data is being piped to stdin")

		peerAddresses = getPeerAddressesFromFile(os.Stdin)
	} else {
		log.Println("stdin is from a terminal")

		peerAddressesFlag := flag.String("peers", "", "semicolon seperated list peer addresses")
		flag.Parse()
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
	address, err := gossipNode.Start()
	if err != nil {
		panic(err)
	}

	//select 4 peers only
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(peerAddresses), func(i, j int) { peerAddresses[i], peerAddresses[j] = peerAddresses[j], peerAddresses[i] })
	for index, address := range peerAddresses {

		if index == appConfig.Network.PeerCount {
			break
		}

		connectToPeer(address, gossipNode)
	}

	app.Start()
	fmt.Println(address)

	log.Println(address)

	gossipNode.Wait()

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
