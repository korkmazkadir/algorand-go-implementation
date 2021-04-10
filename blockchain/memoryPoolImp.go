package blockchain

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type memoryPoolImp struct {
	blockPayloadSize int
	transactions     map[string]Transaction
	log              *log.Logger
}

// NewMemoryPool creates a memorypool
func NewMemoryPool(blockPayloadSize int, logger *log.Logger) MemoryPool {
	mp := new(memoryPoolImp)
	mp.blockPayloadSize = blockPayloadSize
	mp.transactions = make(map[string]Transaction)
	mp.log = logger

	mp.log.Println("New memory pool created.")
	return mp
}

func (mp *memoryPoolImp) AddTransaction(tx Transaction) {
	panic(fmt.Errorf("operation does not supported"))
}

func (mp *memoryPoolImp) RemoveTransactions(block Block) {
	panic(fmt.Errorf("operation does not supported"))
}

// CreateBlock creates a block with random payload with maxBlockPayloadSize
func (mp *memoryPoolImp) CreateBlock(previousBlockHash []byte, blockIndex int) *Block {

	block := Block{}
	block.Timestamp = time.Now().Unix()
	block.PrevHash = previousBlockHash
	block.Index = blockIndex

	rand.Seed(time.Now().UnixNano())
	// to decrease the memory consumption, in memory we will keep only 256 byte
	// random data. We will append the extra data before sending message over the network
	// I hope this will decrease memory consumption of 100 nodes on a machine
	// payload := make([]byte, mp.blockPayloadSize)
	payload := make([]byte, 256)

	size, err := rand.Read(payload)
	if err != nil || size != mp.blockPayloadSize {
		panic(fmt.Errorf("could not create random payload for block. %s size %d", err, size))
	}

	block.Transactions = payload
	block.TxRootHash = digest(block.Transactions)

	mp.log.Printf("New block created. Payload Size: %d Bytes \n", len(block.Transactions))

	return &block
}

func (mp *memoryPoolImp) CreateEmptyBlock(previousBlock *MacroBlock, blockIndex int) *Block {

	emptyBlock := new(Block)
	emptyBlock.PrevHash = previousBlock.Hash()
	emptyBlock.Index = blockIndex
	emptyBlock.SeedHash = previousBlock.SeedHash()

	return emptyBlock
}

func (mp *memoryPoolImp) GetPayloadSize() int {
	// 256 bytes is already appended to the paylaod of a created block
	return mp.blockPayloadSize - 256
}
