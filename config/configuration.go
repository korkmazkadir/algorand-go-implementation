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
	TBigFinal         int
	TSmallFinal       float32

	LamdaPriority int
	LamdaBlock    int
	LamdaStep     int
	LamdaStepVar  int
}

type BlockchainPrameters struct {
	BlockPayloadSize int
	//RemoveOldBlocks          bool
	//RemovePayloadAfterAppend bool
}

type NetworkParameters struct {
	GossipNodeMessageBufferSize int
	PeerCount                   int
}

type LogParameters struct {
	EnableAgreementLoging  bool
	EnableMemoryPoolLoging bool
	EnableBlockchainLoging bool
	EnableGossipNodeLoging bool
}

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
