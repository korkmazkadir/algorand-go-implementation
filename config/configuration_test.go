package config

import (
	"encoding/json"
	"log"
	"testing"
)

func TestConfiguration(t *testing.T) {

	appConfig := readConfigurationFromFile()

	if appConfig.BAStar.ConcurrencyConstant != 3 {
		t.Errorf("could not read concurrency constant...")
	}

}

func readConfigurationFromFile() *Configuration {
	fileName := "../config.json"
	appConfig := ReadConfigurationFromFile(fileName)
	appConfig.BAStar.TotalMoney = uint64(appConfig.NodeCount) * (appConfig.BAStar.UserMoney)

	configData, _ := json.MarshalIndent(appConfig, "", "\t")
	log.Printf("Starting node with configuration \n %s \n", string(configData))
	return appConfig
}
