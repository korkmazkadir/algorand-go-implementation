package blockchain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
)

type blockchainImp struct {
	blocks        []Block
	lastBlock     *Block
	lastBlockHash []byte
	blockHeight   int
	log           *log.Logger
}

// NewBlockchain creates a blockchain with the predefined genesis block
func NewBlockchain(logger *log.Logger) Blockchain {

	blockChain := new(blockchainImp)
	blockChain.blocks = make([]Block, 0)

	genesisBlock := Block{
		Timestamp: 0,
		Issuer:    []byte{128, 33, 77, 11, 96, 197, 158},
		Index:     0,
	}
	genesisBlock.SeedHash = digest([]byte("07.11.2020 I am here"))
	genesisHash := genesisBlock.Hash()
	blockChain.lastBlockHash = genesisHash
	blockChain.blocks = append(blockChain.blocks, genesisBlock)
	blockChain.blockHeight = 1
	blockChain.lastBlock = &genesisBlock
	blockChain.log = logger

	blockChain.log.Println("New blochain created.")
	blockChain.log.Printf("Genesis block created: %s", base64.StdEncoding.EncodeToString(genesisHash))

	return blockChain
}

func (b *blockchainImp) AppendBlock(block Block) error {

	if block.Index != b.blockHeight {
		return fmt.Errorf("current blockchain height %d is not equal to block index %d", b.blockHeight, block.Index)
	}

	if bytes.Equal(block.PrevHash, b.lastBlockHash) == false {
		return errors.New("previous block hash is not equal to the hash of the last appende block")
	}

	blockHash := block.Hash()
	b.blocks = append(b.blocks, block)
	b.lastBlock = &block
	// update hight proporley!!!
	b.blockHeight = b.blockHeight + 1
	b.lastBlockHash = blockHash

	//Removes payload of the block to use less memory!!!!
	block.Transactions = nil

	//Removes the previously appended block to use less memory!!!
	if len(b.blocks) == 2 {
		b.blocks = b.blocks[1:]
	}

	b.log.Printf("New block appended: %s\n", base64.StdEncoding.EncodeToString(blockHash))
	b.log.Printf("Blockchain contains  %d blocks\n", len(b.blocks))

	return nil
}

func (b *blockchainImp) GetBlockHeight() int {
	return b.blockHeight
}

func (b *blockchainImp) GetLastBlock() *Block {
	return b.lastBlock
}

func (b *blockchainImp) GetLastBlockHash() []byte {
	return b.lastBlockHash
}

func (b *blockchainImp) GetLastBlockSeedHash() []byte {
	return b.lastBlock.SeedHash
}
