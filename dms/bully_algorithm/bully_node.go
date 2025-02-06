package bully

import (
	"log"
	"net"
	"net/rpc"
	"time"

	"github.com/LK4D4/trylock"
	"github.com/grussorusso/serverledge/dms/bully_algorithm/event"
)

// nodeAddressByID: It includes nodes currently in cluster TODO: it must not be hardcoded, get them from registry
var nodeAddressByID = map[string]string{
	"node-01": "node-01:6001",
	"node-02": "node-02:6002",
	"node-03": "node-03:6003",
	"node-04": "node-04:6004",
}

type NodeInfo struct {
	Status        string        // Normal, Degraded, Critical, Inactive
	CloudDist     time.Duration // time distance, calculated with vivaldi
	AvailableRsrc float64       // percentage on RAM, CPU and Net I/O
	SuperBully    bool          // if true the node has to be the master
}

var InfoRwMtx trylock.Mutex
var ElectionUpdate chan string

func (i NodeInfo) hasCloudNeighbour() bool {
	// fmt.Printf("Hello, my name is %s and I am %d years old.\n", i.status, i.cloudDist)
	if i.CloudDist != 0 {
		return true
	}
	return false
}

type BullyNode struct {
	ID       string
	Addr     string
	Info     NodeInfo
	Peers    *Peers
	eventBus event.Bus
}

func NewBullyNode(nodeID string) *BullyNode {
	node := &BullyNode{
		ID:   nodeID,
		Addr: nodeAddressByID[nodeID],
		// add info
		Peers:    NewPeers(),
		eventBus: event.NewBus(),
	}

	node.eventBus.Subscribe(event.LeaderElected, node.PingLeaderContinuously)

	return node
}

func (node *BullyNode) NewListener() (net.Listener, error) {
	addr, err := net.Listen("tcp", node.Addr)
	return addr, err
}

func (node *BullyNode) ConnectToPeers() {
	for peerID, peerAddr := range nodeAddressByID {
		if node.IsItself(peerID) {
			continue
		}

		rpcClient := node.connect(peerAddr)
		pingMessage := Message{FromPeerID: node.ID, Type: PING}
		reply, _ := node.CommunicateWithPeer(rpcClient, pingMessage)

		if reply.IsPongMessage() {
			log.Println("%s got pong message from %s", node.ID, peerID)
			node.Peers.Add(peerID, rpcClient, reply.info)
		}
	}
}

func (node *BullyNode) connect(peerAddr string) *rpc.Client {
retry:
	client, err := rpc.Dial("tcp", peerAddr)
	if err != nil {
		log.Println("Error dialing rpc dial %s", err.Error())
		time.Sleep(50 * time.Millisecond)
		goto retry
	}
	return client
}

func (node *BullyNode) CommunicateWithPeer(RPCClient *rpc.Client, args Message) (Message, error) {
	var reply Message

	err := RPCClient.Call("BullyNode.HandleMessage", args, &reply)
	if err != nil {
		log.Println("Error calling HandleMessage %s", err.Error())
	}

	return reply, err
}

func (node *BullyNode) HandleMessage(args Message, reply *Message) error {
	reply.FromPeerID = node.ID

	switch args.Type {
	case ELECTION:
		reply.Type = ALIVE
	case ELECTED:
		leaderID := args.FromPeerID
		log.Println("Election is done. %s has a new leader %s", node.ID, leaderID)
		node.eventBus.Emit(event.LeaderElected, leaderID)
		reply.Type = OK
	case PING:
		reply.Type = PONG
	}

	return nil
}

func (node *BullyNode) Elect(update chan string) {
	isHighestRankedBullyNodeAvailable := false
	var leaderID string
	peers := node.Peers.ToList()
	for i := range peers {
		peer := peers[i]

		if node.Info.SuperBully {
			// force to be the leader
			continue
		}

		if node.IsRankedHigherThan(peer) {
			continue
		}

		log.Println("%s send ELECTION message to peer %s", node.ID, peer.ID)
		electionMessage := Message{FromPeerID: node.ID, Type: ELECTION}

		reply, _ := node.CommunicateWithPeer(peer.RPCClient, electionMessage)

		if reply.IsAliveMessage() {
			isHighestRankedBullyNodeAvailable = true
			leaderID = peer.ID
		}
	}

	if node.Info.SuperBully {
		// force to be the leader
		leaderID = node.ID
		electedMessage := Message{FromPeerID: leaderID, Type: ELECTED}
		node.BroadcastMessage(electedMessage)
		log.Println("%s is a new leader", node.ID)
		update <- leaderID
		return
	}

	if !isHighestRankedBullyNodeAvailable {
		leaderID = node.ID
		electedMessage := Message{FromPeerID: leaderID, Type: ELECTED}
		node.BroadcastMessage(electedMessage)
		log.Println("%s is a new leader", node.ID)
		update <- leaderID
		return
	}

	update <- leaderID
}

func (node *BullyNode) BroadcastMessage(args Message) {
	peers := node.Peers.ToList()
	for i := range peers {
		peer := peers[i]
		node.CommunicateWithPeer(peer.RPCClient, args)
	}
}

func (node *BullyNode) PingLeaderContinuously(_ string, payload any) {
	leaderID := payload.(string)

ping:
	leader := node.Peers.Get(leaderID)
	if leader == nil {
		log.Println("%s, %s, %s", node.ID, leaderID, node.Peers.ToIDs())
		return
	}

	pingMessage := Message{FromPeerID: node.ID, Type: PING}
	reply, err := node.CommunicateWithPeer(leader.RPCClient, pingMessage)
	if err != nil {
		log.Println("Leader is down, new election about to start!")
		node.Peers.Delete(leaderID)
		node.Elect(ElectionUpdate)
		return
	}

	if reply.IsPongMessage() {
		log.Println("Leader %s sent PONG message", reply.FromPeerID)
		time.Sleep(3 * time.Second)
		goto ping
	}
}

// func (node *BullyNode) IsRankHigherThan(id string) bool {
// 	return strings.Compare(node.ID, id) == 1
// }

// new implementation
func (node *BullyNode) IsRankedHigherThan(peer Peer) bool {
	thisNode := node.Info
	if thisNode.hasCloudNeighbour() && (thisNode.Status == "Normal" || thisNode.Status == "Degraded") {
		// thisNode has the criteria to get elected
		return thisNode.checkElectionConditions(node.ID, peer)
	} else {
		// thisNode has NOT the criteria to get elected
		return false //thisNode.acceptOthersElection()
	}
}

func (i NodeInfo) checkElectionConditions(thisID string, other Peer) bool {
	InfoRwMtx.Lock()
	defer InfoRwMtx.Unlock()
	thisNode := i
	otherNode := other.info
	// Primary comparison: status and cloudDist
	if thisNode.Status > otherNode.Status && thisNode.CloudDist < otherNode.CloudDist {
		return true
	}

	// Secondary comparison: same status and cloudDist
	if thisNode.Status == otherNode.Status && thisNode.CloudDist == otherNode.CloudDist {
		if thisNode.AvailableRsrc > otherNode.AvailableRsrc {
			return true
		} else if thisNode.AvailableRsrc == otherNode.AvailableRsrc && thisID > other.ID {
			return true
		} else {
			return false
		}
	}

	// tertiary comparison: same status only
	if thisNode.Status == otherNode.Status {
		if thisNode.AvailableRsrc > otherNode.AvailableRsrc {
			return true
		} else if thisNode.AvailableRsrc == otherNode.AvailableRsrc && thisID > other.ID {
			return true
		} else {
			return false
		}
	}

	// otherNode wins
	return false
}

func (node *BullyNode) IsItself(id string) bool {
	return node.ID == id
}
