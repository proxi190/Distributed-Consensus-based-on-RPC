package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// Server represents an individual server node in the Raft cluster.
type Server struct {
	serverID  int                 // Unique identifier for the server.
	peers     map[int]*rpc.Client // RPC clients for each peer in the cluster.
	rpcServer *rpc.Server         // RPC server to handle incoming requests.
	listener  net.Listener        // Network listener to accept connections.
	mu        sync.Mutex          // Mutex to protect shared resources.
	raft      Raft                // The Raft consensus mechanism.
}

// NewServer initializes a new server with a given ID.
func NewServer(id int, peers []int) *Server {
	s := &Server{
		serverID:  id,
		peers:     make(map[int]*rpc.Client),
		rpcServer: rpc.NewServer(),
		raft:      NewRaftNode(id, peers),
	}
	s.rpcServer.RegisterName("Raft", s.raft)
	return s
}

// Start initializes the server's RPC system and starts listening on the specified port.
func (s *Server) Start(port int) error {
	serverAddress := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return err
	}
	s.listener = lis
	log.Printf("Server started on %s\n", serverAddress)

	// Register the server and its peers
	peerIDs := s.extractPeerIDs()
	for _, pid := range peerIDs {
		s.ConnectToPeer(pid, fmt.Sprintf("127.0.0.1:%d", 8000+pid))
	}

	// Handle incoming connections in a new goroutine.
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
			go s.rpcServer.ServeConn(conn)
		}
	}()

	return nil
}

// extractPeerIDs returns a slice of peer IDs from the peers map.
func (s *Server) extractPeerIDs() []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]int, 0, len(s.peers))
	for id := range s.peers {
		ids = append(ids, id)
	}
	return ids
}

// ConnectToPeer establishes an RPC client connection to another server.
func (s *Server) ConnectToPeer(peerID int, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.peers[peerID]; ok {
		return fmt.Errorf("already connected to peer %d", peerID)
	}

	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return err
	}
	s.peers[peerID] = client
	log.Printf("Connected to peer %d at %s", peerID, address)

	return nil
}

func (s *Server) DisconnectPeer(peerID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.peers[peerID]; ok {
		client.Close()
		delete(s.peers, peerID)
		log.Printf("[%s] Disconnected from peer %d", time.Now().Format(time.RFC3339), peerID)
		return nil
	}
	return fmt.Errorf("not connected to peer %d", peerID)
}

func (s *Server) Shutdown() {
	s.listener.Close()
	for _, client := range s.peers {
		client.Close()
	}
	log.Printf("[%s] Server shutdown completed", time.Now().Format(time.RFC3339))
}
