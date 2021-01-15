package agreement

import (
	"fmt"
	"log"
	"sync"

	"../blockchain"
	"github.com/korkmazkadir/go-rpc-node/node"
)

const blockQueueSize = 100
const voteQueueSize = 1000

type waitFunction func()

type demux struct {

	//current round
	currentRound int

	// round - vote channel
	voteChanMap map[int]map[string]chan incommingVote

	// block - vote channel
	blockChanMap map[int]chan incommingBlock

	// round - proposal channel
	proposalChanMap map[int]chan incommingProposal

	// round - received message hash
	receivedMessageMap map[int]map[string]struct{}

	mutex sync.Mutex
}

// NewDemux creates and returns a demux instance
func newDemux(currentRound int) *demux {
	d := new(demux)
	d.currentRound = currentRound
	d.voteChanMap = make(map[int]map[string]chan incommingVote)
	d.blockChanMap = make(map[int]chan incommingBlock)
	d.proposalChanMap = make(map[int]chan incommingProposal)
	d.receivedMessageMap = make(map[int]map[string]struct{})
	d.mutex = sync.Mutex{}
	return d
}

func (d *demux) EnqueueMessage(message node.Message) {

	switch message.Tag {
	case tagBlock:

		block := blockchain.Block{}
		node.DecodeFromByte(message.Payload, &block)
		inBlock := incommingBlock{block: block, forward: message.Forward}
		wait, result := d.enqueueBlock(inBlock, message.Hash())
		if result == false {
			log.Printf("waiting to enqueue a block. Round: %d hash: %s\n", block.Index, ByteToBase64String(block.Hash()))
			wait()
		}

	case tagVote:

		vote := Vote{}
		node.DecodeFromByte(message.Payload, &vote)
		inVote := incommingVote{vote: vote, forward: message.Forward}
		wait, result := d.enqueueVote(inVote, message.Hash())
		if result == false {
			log.Printf("Waiting to enqueue a vote. Round %d\n", vote.Round)
			wait()
		}

	case tagProposal:
		proposal := Proposal{}
		node.DecodeFromByte(message.Payload, &proposal)
		inProposal := incommingProposal{proposal: proposal, forward: message.Forward}
		wait, result := d.enqueueProposal(inProposal, message.Hash())
		if result == false {
			log.Printf("Waiting to enqueue a proposal. Round %d\n", proposal.Index)
			wait()
		}

	default:
		panic(fmt.Errorf("Unknow message tag for BAStar protocol: %s", message.Tag))
	}

}

func (d *demux) SetRound(round int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.currentRound = round

	for r := range d.blockChanMap {
		if r < d.currentRound {
			delete(d.blockChanMap, r)
		}
	}

	for r := range d.voteChanMap {
		if r < d.currentRound {
			delete(d.voteChanMap, r)
		}
	}

	for r := range d.proposalChanMap {
		if r < d.currentRound {
			delete(d.proposalChanMap, r)
		}
	}

	for r := range d.receivedMessageMap {
		if r < d.currentRound {
			delete(d.receivedMessageMap, r)
		}
	}

}

func (d *demux) GetProposalChan(round int) chan incommingProposal {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if round != d.currentRound {
		panic(fmt.Errorf("Current round %d not equals provided round %d", d.currentRound, round))
	}

	_, ok := d.proposalChanMap[round]
	if ok == false {
		d.proposalChanMap[round] = createProposalChan()
	}

	return d.proposalChanMap[round]
}

func (d *demux) GetBlockChan(round int) chan incommingBlock {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if round != d.currentRound {
		panic(fmt.Errorf("Current round %d not equals provided round %d", d.currentRound, round))
	}

	_, ok := d.blockChanMap[round]
	if ok == false {
		d.blockChanMap[round] = createBlockChan()
	}

	return d.blockChanMap[round]
}

func (d *demux) GetVoteChan(round int, step string) chan incommingVote {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if round != d.currentRound {
		panic(fmt.Errorf("Current round %d not equals provided round %d", d.currentRound, round))
	}

	_, ok := d.voteChanMap[round]
	if ok == false {
		d.voteChanMap[round] = createVoteStepMap()
	}

	_, ok = d.voteChanMap[round][step]
	if ok == false {
		d.voteChanMap[round][step] = createVoteChan()
	}

	return d.voteChanMap[round][step]
}

func (d *demux) enqueueBlock(ib incommingBlock, messageHash string) (waitFunction, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//discards the message
	if ib.block.Index < d.currentRound {
		log.Printf(">>> discarding block %s \n", ByteToBase64String(ib.block.Hash()))
		return nil, true
	}

	//The message is already enqueued so no need to enqueue again
	if d.alreadyEnqueued(ib.block.Index, messageHash) {
		return nil, true
	}
	d.markAsEnqueued(ib.block.Index, messageHash)

	_, ok := d.blockChanMap[ib.block.Index]
	if ok == false {
		d.blockChanMap[ib.block.Index] = createBlockChan()
	}

	// enques the block

	select {
	case d.blockChanMap[ib.block.Index] <- ib:
		return nil, true
	default:
		log.Printf("WARNING: Could not enqueue the block %s \n", ByteToBase64String(ib.block.Hash()))
		waitFunc := func() {
			d.blockChanMap[ib.block.Index] <- ib
			log.Printf("Late block enqueue. Round: %d Hash: %s\n", ib.block.Index, ByteToBase64String(ib.block.Hash()))
		}

		return waitFunc, false
	}

}

func (d *demux) enqueueVote(iv incommingVote, messageHash string) (waitFunction, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//discards the message
	if iv.vote.Round < d.currentRound {
		return nil, true
	}

	//The message is already enqueued so no need to enqueue again
	if d.alreadyEnqueued(iv.vote.Round, messageHash) {
		return nil, true
	}
	d.markAsEnqueued(iv.vote.Round, messageHash)

	_, ok := d.voteChanMap[iv.vote.Round]
	if ok == false {
		d.voteChanMap[iv.vote.Round] = createVoteStepMap()
	}

	_, ok = d.voteChanMap[iv.vote.Round][iv.vote.Step]
	if ok == false {
		d.voteChanMap[iv.vote.Round][iv.vote.Step] = createVoteChan()
	}

	// enques the vote
	select {
	case d.voteChanMap[iv.vote.Round][iv.vote.Step] <- iv:
		return nil, true
	default:
		log.Println("WARNING: Could not enqueue the vote\n")
		waitFunc := func() {
			d.voteChanMap[iv.vote.Round][iv.vote.Step] <- iv
			log.Printf("Late vote enqueue. Round %d \n", iv.vote.Round)
		}

		return waitFunc, false
	}

}

func (d *demux) enqueueProposal(ip incommingProposal, messageHash string) (waitFunction, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//Discards the message
	if ip.proposal.Index < d.currentRound {
		log.Printf(">>> discarding proposal %d \n", ip.proposal.Index)
		return nil, true
	}

	//The message is already enqueued so no need to enqueue again
	if d.alreadyEnqueued(ip.proposal.Index, messageHash) {
		return nil, true
	}
	d.markAsEnqueued(ip.proposal.Index, messageHash)

	_, ok := d.proposalChanMap[ip.proposal.Index]
	if ok == false {
		d.proposalChanMap[ip.proposal.Index] = createProposalChan()
	}

	// enques the proposal
	select {
	case d.proposalChanMap[ip.proposal.Index] <- ip:
		return nil, true
	default:
		log.Println("WARNING: Could not enqueue the proposal %d \n", ip.proposal.Index)
		waitFunc := func() {
			d.proposalChanMap[ip.proposal.Index] <- ip
			log.Printf("Late proposal enqueue. Round: %d Block Hash: %s\n", ip.proposal.Index, ByteToBase64String(ip.proposal.BlockHash))
		}

		return waitFunc, false
	}

}

func (d *demux) alreadyEnqueued(round int, messageHash string) bool {

	_, ok := d.receivedMessageMap[round]
	if ok == false {
		d.receivedMessageMap[round] = make(map[string]struct{})
		return false
	}

	_, ok = d.receivedMessageMap[round][messageHash]

	return ok
}

func (d *demux) markAsEnqueued(round int, messageHash string) {
	d.receivedMessageMap[round][messageHash] = struct{}{}
}

// It is used by the locally to mark a message as already processed
func (d *demux) MarkAsEnqueued(round int, messageHash string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	_, ok := d.receivedMessageMap[round]
	if ok == false {
		d.receivedMessageMap[round] = make(map[string]struct{})
	}

	d.receivedMessageMap[round][messageHash] = struct{}{}
}

func createProposalChan() chan incommingProposal {
	return make(chan incommingProposal, blockQueueSize)
}

func createBlockChan() chan incommingBlock {
	return make(chan incommingBlock, blockQueueSize)
}

func createVoteStepMap() map[string]chan incommingVote {
	return make(map[string]chan incommingVote)
}

func createVoteChan() chan incommingVote {
	return make(chan incommingVote, voteQueueSize)
}
