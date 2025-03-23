package bully

type Message struct {
	FromPeerID string
	Info       NodeInfo // sent only with PONG type message
	Type       MessageType
}

func (m *Message) IsAliveMessage() bool {
	return m.Type == ALIVE
}

func (m *Message) IsPongMessage() bool {
	return m.Type == PONG
}

type MessageType uint32

const (
	PING MessageType = iota + 1
	PONG
	ELECTION
	ALIVE
	ELECTED
	OK
)
