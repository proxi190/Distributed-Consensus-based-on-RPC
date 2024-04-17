package raft

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Debug = 0 // Set to 1 to enable debug output

type LogEntry struct {
	Command   interface{}
	Term      int
	Timestamp time.Time
}

type Raft interface {
	AppendEntry(command interface{}) error
	PerformElectionTimeout()
	// Add other methods as necessary
}

type RaftNode struct {
	id          int
	mu          sync.Mutex
	log         []LogEntry
	state       string
	peers       []*RaftNode
	term        int
	votedFor    int
	currentTerm int
}

// RaftCluster holds the state of the entire cluster.
type RaftCluster struct {
	Nodes []*RaftNode
	mu    sync.Mutex
}

func NewRaftNode(id int, peerIDs []int) *RaftNode {
	node := &RaftNode{
		id:       id,
		log:      make([]LogEntry, 0),
		state:    "follower",
		peers:    make([]*RaftNode, len(peerIDs)),
		term:     0,
		votedFor: -1,
	}
	// Initialize peers within the new node
	for i, pid := range peerIDs {
		if pid != id {
			node.peers[i] = NewRaftNode(pid, peerIDs) // Recursively adds peers excluding self
		}
	}
	return node
}

func (rn *RaftNode) AppendEntry(command interface{}) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	if rn.state != "leader" {
		return errors.New("not the leader")
	}
	rn.log = append(rn.log, LogEntry{Command: command, Term: rn.currentTerm, Timestamp: time.Now()})
	if Debug > 0 {
		fmt.Printf("[%s] Leader Node %d: Appended command to log: %v\n", time.Now().Format(time.RFC3339), rn.id, command)
	}
	return nil
}

// PerformElectionTimeout initiates an election if this node is not a leader.
func (rn *RaftNode) PerformElectionTimeout() {
	rn.mu.Lock()
	if rn.state == "leader" {
		rn.mu.Unlock()
		return // A leader does not initiate an election.
	}

	rn.currentTerm++ // Increment term
	rn.votedFor = rn.id
	rn.state = "candidate"
	rn.mu.Unlock()

	if Debug > 0 {
		fmt.Printf("[%s] Election timeout triggered, node %d transitioning to candidate in term %d\n", time.Now().Format(time.RFC3339), rn.id, rn.currentTerm)
	}

	var votesReceived = 1 // Start with a vote for self
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, peer := range rn.peers {
		wg.Add(1)
		go func(peer *RaftNode) {
			defer wg.Done()
			voteGranted, term := rn.requestVote(peer)
			mu.Lock()
			defer mu.Unlock()
			if voteGranted && term == rn.currentTerm && rn.state == "candidate" {
				votesReceived++
				if votesReceived > len(rn.peers)/2 {
					rn.state = "leader"
					rn.sendHeartbeats()
					if Debug > 0 {
						fmt.Printf("[%s] Node %d became leader for term %d\n", time.Now().Format(time.RFC3339), rn.id, rn.currentTerm)
					}
				}
			}
		}(peer)
	}
	wg.Wait()
}

func (rn *RaftNode) sendHeartbeats() {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	for _, peer := range rn.peers {
		if peer != nil {
			go func(peer *RaftNode) {
				peer.mu.Lock()
				defer peer.mu.Unlock()
				if peer.currentTerm < rn.currentTerm {
					peer.currentTerm = rn.currentTerm // Update term
					peer.state = "follower"           // Convert to follower
				}
			}(peer)
		}
	}
}

// requestVote handles vote requests from this node to a peer.
func (rn *RaftNode) requestVote(peer *RaftNode) (bool, int) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	peer.mu.Lock()
	defer peer.mu.Unlock()

	isLogUpToDate := rn.isLogUpToDate(peer)

	if peer.currentTerm <= rn.currentTerm && (peer.votedFor == -1 || peer.votedFor == rn.id) && isLogUpToDate {
		peer.currentTerm = rn.currentTerm // Update peer's term to the candidate's term
		peer.votedFor = rn.id             // Peer votes for this candidate
		return true, rn.currentTerm
	}
	return false, peer.currentTerm
}

// isLogUpToDate checks if the candidate's log is at least as up-to-date as the peer's log.
func (rn *RaftNode) isLogUpToDate(peer *RaftNode) bool {
	candidateLastLogIndex := len(rn.log) - 1
	peerLastLogIndex := len(peer.log) - 1

	if candidateLastLogIndex < 0 || peerLastLogIndex < 0 {
		return true // Handle edge case where there might be no log entries yet
	}

	candidateLastLogTerm := rn.log[candidateLastLogIndex].Term
	peerLastLogTerm := peer.log[peerLastLogIndex].Term

	return candidateLastLogTerm > peerLastLogTerm || (candidateLastLogTerm == peerLastLogTerm && candidateLastLogIndex >= peerLastLogIndex)
}

func (rc *RaftCluster) SetData(key string, value int) error {
	leader := rc.findLeader()
	if leader == nil {
		return errors.New("no leader found")
	}
	return leader.AppendEntry(fmt.Sprintf("%s=%d", key, value))
}

func (rc *RaftCluster) GetData(key string) (int, error) {
	leader := rc.findLeader()
	if leader == nil {
		return 0, errors.New("no leader found")
	}
	for _, entry := range leader.log {
		parts := strings.Split(entry.Command.(string), "=")
		if len(parts) == 2 && parts[0] == key {
			value, err := strconv.Atoi(parts[1])
			if err != nil {
				return 0, errors.New("error parsing stored value")
			}
			return value, nil
		}
	}
	return 0, errors.New("key not found")
}

func CreateNewCluster(peers int) (*RaftCluster, error) {
	if peers <= 0 {
		return nil, errors.New("invalid number of peers")
	}
	nodes := make([]*RaftNode, peers)
	// First create all nodes
	for i := 0; i < peers; i++ {
		nodes[i] = &RaftNode{
			log:      make([]LogEntry, 0),
			state:    "follower",
			peers:    make([]*RaftNode, 0),
			term:     0,
			votedFor: -1,
		}
	}
	// Then assign peers
	for i := 0; i < peers; i++ {
		for j := 0; j < peers; j++ {
			if i != j {
				nodes[i].peers = append(nodes[i].peers, nodes[j])
			}
		}
	}
	return &RaftCluster{Nodes: nodes}, nil
}

func (rc *RaftCluster) findLeader() *RaftNode {
	for _, node := range rc.Nodes {
		if node.state == "leader" {
			return node
		}
	}
	return nil
}

func (rc *RaftCluster) AddNode(id int) error {
	// Implementation for adding a node to the cluster
	newNode := NewRaftNode(id, nil) // Initialize without peers
	rc.Nodes = append(rc.Nodes, newNode)
	// Update all existing nodes with the new peer
	for _, node := range rc.Nodes {
		node.peers = append(node.peers, newNode)
		newNode.peers = append(newNode.peers, node)
	}
	return nil
}

func (rc *RaftCluster) RemoveNode(id int) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Find the index of the node to remove based on its unique identifier
	nodeIndex := -1
	for i, node := range rc.Nodes {
		if node.id == id { // Assuming each node has an 'id' property that uniquely identifies it
			nodeIndex = i
			break
		}
	}

	if nodeIndex == -1 {
		return errors.New("node not found")
	}

	// Remove the node from the cluster's Nodes slice
	rc.Nodes = append(rc.Nodes[:nodeIndex], rc.Nodes[nodeIndex+1:]...)

	// Update the peers slice of each remaining node
	for i := range rc.Nodes {
		newPeers := make([]*RaftNode, 0)
		for _, peer := range rc.Nodes[i].peers {
			if peer.id != id {
				newPeers = append(newPeers, peer)
			}
		}
		rc.Nodes[i].peers = newPeers
	}

	return nil
}

func (rc *RaftCluster) PrintStatus() {
	for i, node := range rc.Nodes {
		fmt.Printf("Node %d: %s with term %d\n", i, node.state, node.term)
	}
}
