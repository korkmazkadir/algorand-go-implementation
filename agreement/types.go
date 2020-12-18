package agreement

import (
	"bytes"
	"fmt"
)

const (
	StepReductionOne = "REDUCTION_ONE"
	StepReductionTwo = "REDUCTION_TWO"

	Step      = "STEP"
	StepFinal = "STEP_FINAL"

	RoleProposer  = "PROPOSER"
	RoleCommittee = "COMMITTEE"
)

// Vote defines agreement vote structure
type Vote struct {
	Issuer        []byte
	VrfHash       []byte
	VrfProof      []byte
	VoteCount     uint64
	Round         int
	Step          string
	LastBlockHash []byte
	SelectionVector
	//TODO update signature!!!!
	Signature []byte
}

func (v Vote) hashString() string {
	//SenderPK|VrfHash|VrfProof|VoteCount|Round|Step|LastBlockHash|SelectedBlock
	return fmt.Sprintf("%x|%x|%x|%d|%d|%s|%x|%x", v.Issuer, v.VrfHash, v.VrfProof, v.VoteCount, v.Round, v.Step, v.LastBlockHash, v.SelectionVector.Hash())
}

// Hash calculatest he hash of the Vote
func (v Vote) Hash() []byte {
	hashString := v.hashString()
	return digest([]byte(hashString))
}

type context struct {
	publickKey []byte
	privateKey []byte
}

// Proposal defines agreement proposal structure
type Proposal struct {
	Issuer    []byte
	Index     int
	PrevHash  []byte
	SeedProof []byte
	VrfProof  []byte
	BlockHash []byte
	Signature []byte
}

func (p *Proposal) hashString() string {
	//SenderPK|VrfHash|VrfProof|VoteCount|Round|Step|LastBlockHash|SelectedBlock
	return fmt.Sprintf("%x|%d|%x|%x|%x|%x|%x", p.Issuer, p.Index, p.PrevHash, p.SeedProof, p.VrfProof, p.BlockHash)
}

// Hash calculates SHA 256 digest of the proposal
func (p *Proposal) Hash() []byte {
	hashString := p.hashString()
	return digest([]byte(hashString))
}

// SelectionVector defines
type SelectionVector struct {
	//Block hash must be in the correct index
	//Max size is equal to ConcurrencyConstant edfined in the config file
	Hashes    [][]byte
	proposals []*Proposal
	conCons   int
	hash      []byte
}

// NewSelectionVector creates a selection vector
func NewSelectionVector(selectionHashes ...[]byte) SelectionVector {
	return SelectionVector{Hashes: selectionHashes}
}

func NewSelectionVectorWithSize(concurrencyConstant int) SelectionVector {
	hashes := [][]byte{}
	proposals := []*Proposal{}

	for i := 0; i < concurrencyConstant; i++ {
		hashes = append(hashes, nil)
		proposals = append(proposals, nil)
	}

	return SelectionVector{Hashes: hashes, proposals: proposals, conCons: concurrencyConstant}
}

// Hash calculates combined hash of the selection vector
func (sv *SelectionVector) Hash() []byte {
	if sv.hash == nil {
		sv.hash = CombinedHash(sv.Hashes)
	}
	return sv.hash
}

func (sv *SelectionVector) Add(proposal Proposal) bool {

	blockIndex := CalculateBlockIndex(proposal.VrfProof, sv.conCons)
	if sv.proposals[blockIndex] == nil || compareProposals(sv.proposals[blockIndex], &proposal) < 0 {
		sv.proposals[blockIndex] = &proposal
		sv.Hashes[blockIndex] = proposal.BlockHash
		return true
	}

	return false
}

func (sv *SelectionVector) Size() int {

	counter := 0
	for _, blockHash := range sv.Hashes {
		if blockHash != nil {
			counter++
		}
	}
	return counter
}

// SelectsEmptyBlock returns true if selection vector contains only the empty block hash
func (sv *SelectionVector) SelectsEmptyBlock(emptyBlockHash []byte) bool {
	if len(sv.Hashes) != 1 {
		return false
	}

	return bytes.Equal(sv.Hashes[0], emptyBlockHash)
}
