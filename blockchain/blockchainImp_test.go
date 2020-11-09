package blockchain

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func TestBlockchainImp(t *testing.T) {

	maxBlockPayloadSize := 1000

	flags := log.Ldate | log.Ltime | log.Lmsgprefix
	blockchainLogger := log.New(os.Stderr, "[blockchain] ", flags)
	memoryPoolLogger := log.New(os.Stderr, "[memor-pool] ", flags)

	blockchain := NewBlockchain(blockchainLogger)
	memoryPool := NewMemoryPool(maxBlockPayloadSize, memoryPoolLogger)

	blockHeight := blockchain.GetBlockHeight()
	if blockHeight != 1 {
		t.Errorf("block height must be 1 but returned %d", blockHeight)
	}

	lastBlockHash := blockchain.GetLastBlockHash()
	block := memoryPool.CreateBlock(lastBlockHash, blockHeight)

	if len(block.Transactions) != maxBlockPayloadSize {
		t.Errorf("wrong paylaod size")
	}

	err := blockchain.AppendBlock(block)
	if err != nil {
		t.Errorf("could not append block to blockchain %d", err)
	}

	blockHeight = blockchain.GetBlockHeight()
	if blockHeight != 2 {
		t.Errorf("block height must be 2 but returned %d", blockHeight)
	}

	lastBlockHash2 := blockchain.GetLastBlockHash()
	if bytes.Equal(lastBlockHash, lastBlockHash2) {
		t.Errorf("two last block hash must be different")
	}

	err = blockchain.AppendBlock(block)
	if err == nil {
		t.Errorf("appended same block twice")
	}

	block = memoryPool.CreateBlock(lastBlockHash, blockHeight)
	err = blockchain.AppendBlock(block)
	if err == nil {
		t.Errorf("appended a different block but with a same previous hash")
	}

	block = memoryPool.CreateBlock(lastBlockHash2, blockHeight)
	err = blockchain.AppendBlock(block)
	if err != nil {
		t.Errorf("could not append a correct block")
	}

	blockHeight = blockchain.GetBlockHeight()
	if blockHeight != 3 {
		t.Errorf("block height must be 3 but returned %d", blockHeight)
	}

}
