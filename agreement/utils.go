package agreement

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"

	"../blockchain"
)

func digest(input []byte) []byte {
	h := sha256.New()
	h.Write(input)
	return h.Sum(nil)
}

// ByteToBase64String converts a byte slice to a base64 string
func ByteToBase64String(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func compareBlocks(a *blockchain.Block, b *blockchain.Block) int {

	hashA := a.Hash()
	hashB := b.Hash()

	return bytes.Compare(hashA, hashB)
}

func signBlock(block *blockchain.Block, privateKey []byte) {
	blockHash := block.Hash()
	block.Signature = ed25519.Sign(privateKey, blockHash)
}

func signVote(vote *Vote, privateKey []byte) {
	voteHash := vote.Hash()
	vote.Signature = ed25519.Sign(privateKey, voteHash)
}

func isProposalSignatureValid(proposal *Proposal) bool {
	proposalHash := proposal.Hash()
	return ed25519.Verify(proposal.Issuer, proposalHash, proposal.Signature)
}

func isBlockSignatureValid(block *blockchain.Block) bool {
	blockHash := block.Hash()
	return ed25519.Verify(block.Issuer, blockHash, block.Signature)
}

func isVoteSignatureValid(vote *Vote) bool {
	voteHash := vote.Hash()
	return ed25519.Verify(vote.Issuer, voteHash, vote.Signature)
}

func createProposal(block *blockchain.Block) *Proposal {

	proposal := Proposal{
		Issuer:    block.Issuer,
		Index:     block.Index,
		PrevHash:  block.PrevHash,
		SeedProof: block.SeedProof,
		VrfProof:  block.VrfProof,
		BlockHash: block.Hash(),
	}

	return &proposal
}

func signProposal(proposal *Proposal, privateKey []byte) {
	proposalHash := proposal.Hash()
	proposal.Signature = ed25519.Sign(privateKey, proposalHash)
}

// TODO: use sub user Index as described in the paper!!!
func compareProposals(a *Proposal, b *Proposal) int {

	hashA := digest(a.VrfProof)
	hashB := digest(b.VrfProof)

	return bytes.Compare(hashA, hashB)
}

// TODO: use sub user Index as described in the paper!!!
func compareProposalWithBlock(a *Proposal, b *blockchain.Block) int {

	hashA := digest(a.VrfProof)
	hashB := digest(b.VrfProof)

	return bytes.Compare(hashA, hashB)
}

// XORBytes takes two byte slices and XORs them together, returning the final
// byte slice. It is an error to pass in two byte slices that do not have the
// same length.
// https://github.com/hashicorp/vault/blob/v1.6.1/helper/xor/xor.go#L13
func XORBytes(a, b []byte) ([]byte, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("length of byte slices is not equivalent: %d != %d", len(a), len(b))
	}

	buf := make([]byte, len(a))

	for i := range a {
		buf[i] = a[i] ^ b[i]
	}

	return buf, nil
}

// CombinedHash calculates a combined hash
func CombinedHash(blockHashes [][]byte) []byte {

	var err error
	combinedHash := blockHashes[0]
	for i := 1; i < len(blockHashes); i++ {
		if blockHashes[i] == nil {
			continue
		}

		combinedHash, err = XORBytes(combinedHash, blockHashes[i])
		if err != nil {
			panic(fmt.Errorf("xor byte slice error: ", err))
		}
	}

	return combinedHash
}

// GetMissingBlocks calculates missing block list, if all blocks are aveilable returns nil
func GetMissingBlocks(selection SelectionVector, receivedBlocks []blockchain.Block) [][]byte {

	missingBlocks := [][]byte{}

	for _, blockHash := range selection.Hashes {
		if BlockAvailable(blockHash, receivedBlocks) == false {
			missingBlocks = append(missingBlocks, blockHash)
		}
	}

	if len(missingBlocks) == 0 {
		return nil
	}

	return missingBlocks
}

// ConstructMacroBlock creates a macro block to append the blockchain
func ConstructMacroBlock(selection SelectionVector, receivedBlocks []blockchain.Block) *blockchain.MacroBlock {
	blocks := []blockchain.Block{}

	for _, blockHash := range selection.Hashes {
		for _, block := range receivedBlocks {
			if bytes.Equal(blockHash, block.Hash()) {
				blocks = append(blocks, block)
			}
		}
	}

	return blockchain.NewMacroBlock(blocks)
}

// BlockAvailable returns false if slice doesnot contain the block with a specific hash
func BlockAvailable(blockHash []byte, receivedBlocks []blockchain.Block) bool {

	for _, b := range receivedBlocks {
		if bytes.Equal(b.Hash(), blockHash) {
			return true
		}
	}

	return false
}

func CalculateBlockIndex(vrfProof []byte, concurrencyConstant int) int {
	t := &big.Int{}
	t.SetBytes(digest(vrfProof))

	c := &big.Int{}
	c.SetInt64(int64(concurrencyConstant))

	return int(t.Mod(t, c).Uint64())
}
