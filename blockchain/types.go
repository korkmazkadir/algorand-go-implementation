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
	Signature    []byte
	TxRootHash   []byte
	Transactions []byte
	hash         []byte
}

// Hash calculates SHA256 digest of the block
func (b *Block) Hash() []byte {

	if b.hash == nil {
		text := fmt.Sprintf("%d|%x|%d|%x|%x|%x|%x|%x|%x", b.Timestamp, b.Issuer, b.Index, b.PrevHash, b.SeedHash, b.SeedProof, b.VrfHash, b.VrfProof, b.TxRootHash)
		b.hash = digest([]byte(text))
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
}

// Blockchain defines the interface of blockchain
type Blockchain interface {
	AppendBlock(block Block) error
	GetBlockHeight() int

	GetLastBlock() *Block
	GetLastBlockHash() []byte
	GetLastBlockSeedHash() []byte
}
