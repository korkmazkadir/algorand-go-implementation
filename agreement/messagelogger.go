package agreement

import (
	"log"
)

// Keeps counts of forwarded messages per round
// not thread safe
type messageCountLogger struct {
	round         int
	blockCount    int
	proposalCount int
	voteCount     int
}

func (m *messageCountLogger) setRound(round int) {
	// prints a log line round - block count - proposal count - vote count
	log.Printf("[messages]\t%d\t%d\t%d\t%d\n", m.round, m.blockCount, m.proposalCount, m.voteCount)

	m.round = round
	m.blockCount = 0
	m.proposalCount = 0
	m.voteCount = 0
}

func (m *messageCountLogger) forwardingBlock() {
	m.blockCount++
}

func (m *messageCountLogger) forwardingProposal() {
	m.proposalCount++
}

func (m *messageCountLogger) forwardingVote() {
	m.voteCount++
}
