package raft

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ClusterSimulator manages a cluster of Raft servers for simulation purposes.
type ClusterSimulator struct {
	mu       sync.Mutex
	servers  map[int]*Server       // Maps server IDs to server instances.
	networks map[int]chan struct{} // Channels to simulate network partitions.
}

// NewClusterSimulator initializes a new ClusterSimulator with the given number of nodes.
func NewClusterSimulator(numNodes int) *ClusterSimulator {
	cs := &ClusterSimulator{
		servers:  make(map[int]*Server),
		networks: make(map[int]chan struct{}),
	}
	for i := 0; i < numNodes; i++ {
		peerIDs := make([]int, 0)
		for j := 0; j < numNodes; j++ {
			if i != j {
				peerIDs = append(peerIDs, j)
			}
		}
		server := NewServer(i, peerIDs)
		cs.servers[i] = server
		cs.networks[i] = make(chan struct{}, 1) // Buffered to avoid blocking.
	}
	return cs
}

// StartAllServers initializes and starts all servers in the simulator.
func (cs *ClusterSimulator) StartAllServers() {
	for id, server := range cs.servers {
		go func(sid int, s *Server) {
			err := s.Start(int(8000 + sid)) // Example port numbering scheme.
			if err != nil {
				log.Printf("Error starting server %d: %v", sid, err)
			}
		}(id, server)
	}
}

// StopAllServers gracefully shuts down all servers in the simulator.
func (cs *ClusterSimulator) StopAllServers() {
	for _, server := range cs.servers {
		server.Shutdown()
	}
}

// AddServer simulates adding a new server to the cluster.
func (cs *ClusterSimulator) AddServer(serverID int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	peerIDs := make([]int, 0, len(cs.servers))
	for id := range cs.servers {
		peerIDs = append(peerIDs, id)
	}

	log.Printf("[%s] Adding server with ID %d to the cluster.", time.Now().Format(time.RFC3339), serverID)
	server := NewServer(serverID, peerIDs)
	cs.servers[serverID] = server

	// Establish connections to existing peers
	for _, id := range peerIDs {
		cs.servers[id].ConnectToPeer(serverID, fmt.Sprintf("127.0.0.1:%d", 8000+serverID))
		server.ConnectToPeer(id, fmt.Sprintf("127.0.0.1:%d", 8000+id))
	}

	// Start the server's own RPC server to handle incoming requests
	go func() {
		err := server.Start(8000 + serverID)
		if err != nil {
			log.Printf("[%s] Error starting server %d: %v", time.Now().Format(time.RFC3339), serverID, err)
		} else {
			log.Printf("[%s] Server %d started successfully.", time.Now().Format(time.RFC3339), serverID)
		}
	}()
}

// RemoveServer simulates removing a server from the cluster.
func (cs *ClusterSimulator) RemoveServer(serverID int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	log.Printf("[%s] Removing server %d", time.Now().Format(time.RFC3339), serverID)
	if server, ok := cs.servers[serverID]; ok {
		server.Shutdown()
		delete(cs.servers, serverID)
		for _, s := range cs.servers {
			s.DisconnectPeer(serverID)
		}
	}
}

// Disconnect simulates a network partition by closing the network channel of a server.
func (cs *ClusterSimulator) Disconnect(serverID int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if ch, ok := cs.networks[serverID]; ok {
		close(ch) // Closing the channel simulates a network failure.
	}
}

// Reconnect simulates resolving a network partition by recreating the network channel of a server.
func (cs *ClusterSimulator) Reconnect(serverID int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.networks[serverID]; ok {
		cs.networks[serverID] = make(chan struct{}, 1) // Recreate the channel.
	}
}

// PerformElectionTimeout forces an election timeout to test leadership election.
func (cs *ClusterSimulator) PerformElectionTimeout(serverID int) {
	server := cs.servers[serverID]
	if raftNode, ok := server.raft.(*RaftNode); ok {
		raftNode.PerformElectionTimeout()
	}
}

// PrintStatus prints the status of each server in the cluster.
func (cs *ClusterSimulator) PrintStatus() {
	for id, server := range cs.servers {
		if raftNode, ok := server.raft.(*RaftNode); ok {
			fmt.Printf("Server %d - Term: %d, Leader: %t\n", id, raftNode.currentTerm, raftNode.state == "leader")
		}
	}
}
