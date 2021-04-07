package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type ProtocolParams struct {
	UserMoney  uint64
	TotalMoney uint64

	ThresholdProposer int
	TSmallStep        int
	TBigStep          float32
	TSmallFinal       int
	TBigFinal         float32

	LamdaPriority int
	LamdaBlock    int
	LamdaStep     int
	LamdaStepVar  int

	// macro block size in bytes, micro block size = MacroBlockSize / ConcurrencyConstant
	MacroBlockSize int

	//ConcurrencyConstant is the number of blocks will be appended to blockchain
	ConcurrencyConstant int

	ID int64
}

type BlockchainPrameters struct {
	//BlockPayloadSize int
	//if it is given blockchain stops on the given round
	//0 means do not stop at all!
	StopOnRound int
}

type NetworkParameters struct {
	GossipNodeMessageBufferSize int
	PeerCount                   int
	FanOut                      int
	BigMessageMutex             bool
}

type LogParameters struct {
	EnableAgreementLoging  bool
	EnableMemoryPoolLoging bool
	EnableBlockchainLoging bool
	EnableGossipNodeLoging bool
}

type ValidationParameters struct {
	//if it is false does not validate signatures and VRFs for blocks
	ValidateBlock bool
	//if it is false does not validate signatures and VRFs for votes
	ValidateVote bool
}

// Configuration defines configurable parameters of the program
type Configuration struct {
	//number of nodes in the system
	NodeCount int
	//parameters of the agreement
	BAStar ProtocolParams
	//parameters of the blockchain
	Blockchain BlockchainPrameters
	//prameters of Network
	Network NetworkParameters
	//loging parameters
	Logger LogParameters
	//block and vote validation parameters
	Validation ValidationParameters
}

//ReadConfigurationFromFile reads configuration from a json file
func ReadConfigurationFromFile(fileName string) *Configuration {

	jsonFile, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	config := new(Configuration)
	json.Unmarshal(byteValue, config)

	return config
}

//WriteConfigToFile write configuration to a json file, if file doesnot exist panics
func WriteConfigToFile(config Configuration, fileName string) {

	configData, _ := json.MarshalIndent(config, "", "\t")
	err := ioutil.WriteFile(fileName, configData, 0644)
	if err != nil {
		panic(err)
	}

}
