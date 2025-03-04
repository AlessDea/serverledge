package bully

import (
	"net/rpc"
	"sync"
)

type Peer struct {
	ID        string
	info      NodeInfo
	RPCClient *rpc.Client
}

type ByCDist []Peer

func (a ByCDist) Len() int           { return len(a) }
func (a ByCDist) Less(i, j int) bool { return a[i].info.CloudDist < a[j].info.CloudDist }
func (a ByCDist) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type Peers struct {
	*sync.RWMutex
	peerByID map[string]*Peer
}

func NewPeers() *Peers {
	return &Peers{
		RWMutex:  &sync.RWMutex{},
		peerByID: make(map[string]*Peer),
	}
}

func (p *Peers) Add(ID string, client *rpc.Client, in NodeInfo) {
	p.Lock()
	defer p.Unlock()

	p.peerByID[ID] = &Peer{ID: ID, RPCClient: client, info: in}
}

func (p *Peers) Delete(ID string) {
	p.Lock()
	defer p.Unlock()

	delete(p.peerByID, ID)
}

func (p *Peers) Get(ID string) *Peer {
	p.RLock()
	defer p.RUnlock()

	val := p.peerByID[ID]
	return val
}

func (p *Peers) ToIDs() []string {
	p.RLock()
	defer p.RUnlock()

	peerIDs := make([]string, 0, len(p.peerByID))
	for _, peer := range p.peerByID {
		peerIDs = append(peerIDs, peer.ID)
	}

	return peerIDs
}

func (p *Peers) ToList() []Peer {
	p.RLock()
	defer p.RUnlock()

	peers := make([]Peer, 0, len(p.peerByID))
	for _, peer := range p.peerByID {
		peers = append(peers, *peer)
	}

	return peers
}
