package blockchain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
)

type blockchainImp struct {
	blocks        []MacroBlock
	lastBlock     *MacroBlock
	lastBlockHash []byte
	blockHeight   int
	log           *log.Logger
}

// NewBlockchain creates a blockchain with the predefined genesis block
func NewBlockchain(logger *log.Logger) Blockchain {

	blockChain := new(blockchainImp)
	blockChain.blocks = make([]MacroBlock, 0)

	genesisBlock := Block{
		Timestamp: 0,
		Issuer:    []byte{128, 33, 77, 11, 96, 197, 158},
		Index:     0,
	}

	genesisBlock.SeedHash = digest([]byte("07.11.2020 I am here"))
	genesisMacroBlock := NewMacroBlock([]Block{genesisBlock})
	genesisHash := genesisMacroBlock.Hash()

	blockChain.lastBlockHash = genesisMacroBlock.Hash()
	blockChain.blocks = append(blockChain.blocks, *genesisMacroBlock)
	blockChain.blockHeight = 1
	blockChain.lastBlock = genesisMacroBlock
	blockChain.log = logger

	blockChain.log.Println("New blochain created.")
	blockChain.log.Printf("Genesis block created: %s", base64.StdEncoding.EncodeToString(genesisHash))

	return blockChain
}

func (b *blockchainImp) AppendBlock(block MacroBlock) error {

	if block.index != b.blockHeight {
		return fmt.Errorf("current blockchain height %d is not equal to block index %d", b.blockHeight, block.index)
	}

	if bytes.Equal(block.prevHash, b.lastBlockHash) == false {
		return errors.New("previous block hash is not equal to the hash of the last appende block")
	}

	blockHash := block.Hash()
	b.blocks = append(b.blocks, block)
	b.lastBlock = &block
	// update hight proporley!!!
	b.blockHeight = b.blockHeight + 1
	b.lastBlockHash = blockHash

	b.log.Printf("New block appended: %s\n", base64.StdEncoding.EncodeToString(blockHash))
	for _, mb := range block.microBlocks {
		b.log.Printf("[%s]-------> %s\n", base64.StdEncoding.EncodeToString(blockHash[:10]), base64.StdEncoding.EncodeToString(mb.Hash()[:10]))
	}

	b.log.Printf("Blockchain contains  %d blocks\n", len(b.blocks))

	//Removes payload of the block to use less memory!!!!
	for _, microBlock := range block.microBlocks {
		microBlock.Transactions = nil
	}

	//Removes the previously appended block to use less memory!!!
	if len(b.blocks) == 2 {
		b.blocks = b.blocks[1:]
	}

	return nil
}

func (b *blockchainImp) GetBlockHeight() int {
	return b.blockHeight
}

func (b *blockchainImp) GetLastBlock() *MacroBlock {
	return b.lastBlock
}

func (b *blockchainImp) GetLastBlockHash() []byte {
	return b.lastBlockHash
}

func (b *blockchainImp) GetLastBlockSeedHash() []byte {
	return b.lastBlock.SeedHash()
}
