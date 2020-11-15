package agreement

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"

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
