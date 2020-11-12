package main

import (
	"crypto/ed25519"
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
	"github.com/korkmazkadir/go-rpc-node/node"
)

const totalStake = 1000
const numberOfNodes = 100

const gossipNodeBufferSize = 100

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

	// maxBlockPayloadSize in bytes
	maxBlockPayloadSize := 10

	flags := log.Ldate | log.Ltime | log.Lmsgprefix

	/* Loggers */
	agreementLogger := log.New(os.Stderr, "[agreement] ", flags)
	memoryPoolLogger := log.New(os.Stderr, "[memor-pool] ", flags)
	blockchainLogger := log.New(os.Stderr, "[blockchain] ", flags)

	//ioutil.Discard to discart logs
	nodeLogger := log.New(ioutil.Discard, "[node] ", flags)

	params := agreement.ProtocolParams{

		UserMoney:  totalStake / numberOfNodes,
		TotalMoney: totalStake,

		// expected number of user * per user stake
		TSmallStep: 20 * (totalStake / numberOfNodes),
		//TSmallStep: 2000,
		TBigStep: 0.68,

		// expected number of user * per user stake
		TBigFinal: 30 * (totalStake / numberOfNodes),
		//TBigFinal:   3000,
		TSmallFinal: 0.74,

		ThresholdProposer: 26,

		LamdaPriority: 5,
		LamdaBlock:    60,
		LamdaStep:     20,
		LamdaStepVar:  5,
	}

	memoryPool := blockchain.NewMemoryPool(maxBlockPayloadSize, memoryPoolLogger)
	blockchain := blockchain.NewBlockchain(blockchainLogger)

	pk, sk, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Printf("could not generate key %s", err)
	}

	app := agreement.NewBAStar(params, pk, sk, memoryPool, blockchain, agreementLogger)

	gossipNode := node.NewGossipNode(app, gossipNodeBufferSize, nodeLogger)
	address, err := gossipNode.Start()
	if err != nil {
		panic(err)
	}

	//select 4 peers only
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(peerAddresses), func(i, j int) { peerAddresses[i], peerAddresses[j] = peerAddresses[j], peerAddresses[i] })
	for index, address := range peerAddresses {
		connectToPeer(address, gossipNode)

		if index == 4 {
			break
		}
	}

	app.Start()
	fmt.Println(address)

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
