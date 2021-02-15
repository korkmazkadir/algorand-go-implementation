package blockchain

import (
	"bytes"
	"fmt"
)

// Block defines block
type Block struct {
	Timestamp int64
	Issuer    []byte
	Index     int
	PrevHash  []byte
	Seed
	VrfHash      []byte
	VrfProof     []byte
	TxRootHash   []byte
	Transactions []byte
	hash         []byte
	//----------------//
	Signature []byte
}

func (b Block) hashString() string {
	//Timestamp|Issuer|Index|PrevHash|SeedHash|SeedProof|VrfHash|VrfProof|TxRootHash
	return fmt.Sprintf("%d|%x|%d|%x|%x|%x|%x|%x|%x", b.Timestamp, b.Issuer, b.Index, b.PrevHash, b.SeedHash, b.SeedProof, b.VrfHash, b.VrfProof, b.TxRootHash)
}

// Hash calculates SHA256 digest of the block
func (b *Block) Hash() []byte {

	if b.hash == nil {
		hashString := b.hashString()
		b.hash = digest([]byte(hashString))
	}

	return b.hash
}

// Seed keeps details of seed
type Seed struct {
	// VRF hash
	SeedHash []byte
	// VRF proof
	SeedProof []byte
}

// Transaction defines transaction
type Transaction struct {
	Payload []byte
}

// MemoryPool defines the interface of memory pool
type MemoryPool interface {
	AddTransaction(transaction Transaction)
	//RemoveTransactions(block Block)
	CreateBlock(previousBlockHash []byte, blockIndex int) *Block
	CreateEmptyBlock(previousBlock *MacroBlock, blockIndex int) *Block
}

// Blockchain defines the interface of blockchain
type Blockchain interface {
	AppendBlock(block MacroBlock) error
	GetBlockHeight() int

	GetLastBlock() *MacroBlock
	GetLastBlockHash() []byte
	GetLastBlockSeedHash() []byte
}

// MacroBlock defines block structure for algorand improved
type MacroBlock struct {
	microBlocks []Block
	index       int
	prevHash    []byte
	hash        []byte
	seedHash    []byte
}

// NewMacroBlock creates a new macroblock using block slice
func NewMacroBlock(blocks []Block) *MacroBlock {

	prevHash := blocks[0].PrevHash
	index := blocks[0].Index

	for i := 1; i < len(blocks); i++ {
		if bytes.Equal(prevHash, blocks[i].PrevHash) == false {
			panic("Blocks do not have same previous hash!")
		}
		if index != blocks[i].Index {
			panic("Blocks do not have same Index")
		}
	}

	return &MacroBlock{microBlocks: blocks, index: index, prevHash: prevHash}
}

func (mb *MacroBlock) ifEmptyPanic() {
	if len(mb.microBlocks) == 0 {
		panic("no block is available in the macro block.")
	}
}

// Hash calculates SHA256 digest of the block
func (mb *MacroBlock) Hash() []byte {

	mb.ifEmptyPanic()

	if mb.hash != nil {
		return mb.hash
	}

	var err error
	macroBlockHash := mb.microBlocks[0].Hash()
	for i := 1; i < len(mb.microBlocks); i++ {
		macroBlockHash, err = XORBytes(macroBlockHash, mb.microBlocks[i].Hash())
		if err != nil {
			panic(fmt.Errorf("xor byte slice error: ", err))
		}
	}

	mb.hash = macroBlockHash

	return mb.hash
}

// SeedHash calculates SHA256 digest of the seed
func (mb *MacroBlock) SeedHash() []byte {

	mb.ifEmptyPanic()

	if mb.seedHash != nil {
		return mb.seedHash
	}

	var err error
	macroSeedHash := mb.microBlocks[0].Hash()
	for i := 1; i < len(mb.microBlocks); i++ {
		macroSeedHash, err = XORBytes(macroSeedHash, mb.microBlocks[i].SeedHash)
		if err != nil {
			panic(fmt.Errorf("xor byte slice error: ", err))
		}
	}

	mb.seedHash = macroSeedHash

	return mb.seedHash
}

// PayloadSize returns size of the payload in bytes
func (mb *MacroBlock) PayloadSize() int {
	size := 0
	for _, microBlock := range mb.microBlocks {
		size += len(microBlock.Transactions)
	}
	return size
}
