package agreement

import (
	"crypto/rand"

	"github.com/korkmazkadir/algorand-go-implementation/blockchain"
	"github.com/korkmazkadir/go-rpc-node/node"
)

// applicationImp implements the necessary interface for GosipNode
type applicationImp struct {
	networkReadySig  chan struct{}
	demultiplexer    *demux
	outgoingMessages chan node.Message

	randomPayloadForBlocks []byte
	blockRandomPayloadSize int
}

// it is used to keep the  forward callback
// may be these are not necessary
type incommingProposal struct {
	proposal Proposal
	forward  func()
}

// it is used to keep the  forward callback
// may be these are not necessary
type incommingBlock struct {
	block   blockchain.Block
	forward func()
}

// it is used to keep the  forward callback
// may be these are not necessary
type incommingVote struct {
	vote    Vote
	forward func()
}

// HandleMessage is calld by the GosipNode
func (a *applicationImp) HandleMessage(message node.Message) {

	if message.Tag == tagBlock {
		// It removes the random paylaod from the block to decrease memory consumption
		// I hope it will work :(
		trucatedPaylaod := make([]byte, len(message.Payload)-a.blockRandomPayloadSize)
		copy(trucatedPaylaod, message.Payload)
		message.Payload = trucatedPaylaod
	}

	//It might block
	a.demultiplexer.EnqueueMessage(message)
}

func (a *applicationImp) OutgoingMessageChannel() chan node.Message {
	return a.outgoingMessages
}

func (a *applicationImp) SignalChannel() chan struct{} {
	return a.networkReadySig
}

func (a *applicationImp) BroadcastBlock(block blockchain.Block) {
	payload := node.EncodeToByte(block)
	randomPayload := a.getRandomPayload()
	// adds a random paylaod to create a big message
	message := node.NewMessage(tagBlock, append(payload, randomPayload...))
	a.demultiplexer.MarkAsEnqueued(block.Index, message.Hash())
	a.outgoingMessages <- message
}

func (a *applicationImp) BroadcastProposal(proposal Proposal) {
	payload := node.EncodeToByte(proposal)
	message := node.NewMessage(tagProposal, payload)
	a.demultiplexer.MarkAsEnqueued(proposal.Index, message.Hash())
	a.outgoingMessages <- message
}

func (a *applicationImp) BroadcastVote(vote Vote) {
	payload := node.EncodeToByte(vote)
	message := node.NewMessage(tagVote, payload)
	a.demultiplexer.MarkAsEnqueued(vote.Round, message.Hash())
	a.outgoingMessages <- message
}

func (a *applicationImp) getRandomPayload() []byte {
	if a.randomPayloadForBlocks == nil {
		a.randomPayloadForBlocks = make([]byte, a.blockRandomPayloadSize)
		_, err := rand.Read(a.randomPayloadForBlocks)
		if err != nil {
			panic(err)
		}
	}

	return a.randomPayloadForBlocks
}
