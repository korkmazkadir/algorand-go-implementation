package blockchain

import "fmt"

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
	RemoveTransactions(block Block)
	CreateBlock(previousBlockHash []byte, blockIndex int) *Block
	CreateEmptyBlock(previousBlock *Block, blockIndex int) *Block
}

// Blockchain defines the interface of blockchain
type Blockchain interface {
	AppendBlock(block Block) error
	GetBlockHeight() int

	GetLastBlock() *Block
	GetLastBlockHash() []byte
	GetLastBlockSeedHash() []byte
}
