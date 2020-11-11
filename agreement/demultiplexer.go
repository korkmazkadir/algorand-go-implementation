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

type demux struct {

	//current round
	currentRound int

	// round - vote chanel
	voteChanMap map[int]chan incommingVote

	// block - vote chanel
	blockChanMap map[int]chan incommingBlock

	mutex sync.Mutex
}

// NewDemux creates and returns a demux instance
func newDemux(currentRound int) *demux {
	d := new(demux)
	d.currentRound = currentRound
	d.voteChanMap = make(map[int]chan incommingVote)
	d.blockChanMap = make(map[int]chan incommingBlock)
	d.mutex = sync.Mutex{}
	return d
}

func (d *demux) EnqueueMessage(message node.Message) {

	switch message.Tag {
	case tagBlock:

		block := blockchain.Block{}
		node.DecodeFromByte(message.Payload, &block)
		inBlock := incommingBlock{block: block, forward: message.Forward}
		d.enqueueBlock(inBlock)

	case tagVote:

		vote := Vote{}
		node.DecodeFromByte(message.Payload, &vote)
		inVote := incommingVote{vote: vote, forward: message.Forward}
		d.enqueueVote(inVote)

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

func (d *demux) GetVoteChan(round int) chan incommingVote {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if round != d.currentRound {
		panic(fmt.Errorf("Current round %d not equals provided round %d", d.currentRound, round))
	}

	_, ok := d.voteChanMap[round]
	if ok == false {
		d.voteChanMap[round] = createVoteChan()
	}

	return d.voteChanMap[round]
}

func (d *demux) enqueueBlock(ib incommingBlock) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//discards the message
	if ib.block.Index < d.currentRound {
		log.Printf(">>> discarding block %s \n", ByteToBase64String(ib.block.Hash()))
		return
	}

	_, ok := d.blockChanMap[ib.block.Index]
	if ok == false {
		d.blockChanMap[ib.block.Index] = createBlockChan()
	}

	// enques the block

	select {
	case d.blockChanMap[ib.block.Index] <- ib:
	default:
		log.Println("WARNING: Could not enqueue the block %s \n", ByteToBase64String(ib.block.Hash()))
	}

}

func (d *demux) enqueueVote(iv incommingVote) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//discards the message
	if iv.vote.Round < d.currentRound {
		return
	}

	_, ok := d.voteChanMap[iv.vote.Round]
	if ok == false {
		d.voteChanMap[iv.vote.Round] = createVoteChan()
	}

	select {
	case d.voteChanMap[iv.vote.Round] <- iv:
	default:
		log.Println("WARNING: Could not enqueue the vote\n")
	}

	// enques the vote

}

func createBlockChan() chan incommingBlock {
	return make(chan incommingBlock, blockQueueSize)
}

func createVoteChan() chan incommingVote {
	return make(chan incommingVote, voteQueueSize)
}
