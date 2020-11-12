package agreement

import (
	"../blockchain"
	"../filter"
	"github.com/korkmazkadir/go-rpc-node/node"
)

// applicationImp implements the necessary interface for GosipNode
type applicationImp struct {
	networkReadySig chan struct{}

	messageFilter    *filter.UniqueMessageFilter
	demultiplexer    *demux
	outgoingMessages chan node.Message
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

	isAdded := a.messageFilter.IfNotContainsAdd(message.Hash())
	if isAdded == false {
		// it means that message is already processed!
		return
	}

	//It might blocks
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
	message := node.NewMessage(tagBlock, payload)
	a.outgoingMessages <- message

	a.messageFilter.IfNotContainsAdd(message.Hash())
}

func (a *applicationImp) BroadcastVote(vote Vote) {
	payload := node.EncodeToByte(vote)
	message := node.NewMessage(tagVote, payload)
	a.outgoingMessages <- message

	a.messageFilter.IfNotContainsAdd(message.Hash())
}
