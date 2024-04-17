package raft

import (
	"testing"
	"time"
)

// TestCreateCluster checks the creation of a new cluster and ensures all nodes start as followers.
func TestCreateCluster(t *testing.T) {
	t.Logf("[%s] Starting TestCreateCluster", time.Now().Format(time.RFC3339))
	cluster, err := CreateNewCluster(3)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	for _, node := range cluster.Nodes {
		if node.state != "follower" {
			t.Errorf("Expected node state follower, got %s", node.state)
		}
	}
}

// TestNodeAddition tests the functionality to dynamically add a node to the cluster.
func TestNodeAddition(t *testing.T) {
	t.Logf("[%s] Starting TestNodeAddition", time.Now().Format(time.RFC3339))
	cluster, _ := CreateNewCluster(3)
	err := cluster.AddNode(3) // Adding fourth node
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}
	if len(cluster.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(cluster.Nodes))
	}
}

// TestNodeRemoval tests the functionality to dynamically remove a node from the cluster.
func TestNodeRemoval(t *testing.T) {
	t.Logf("[%s] Starting TestNodeRemoval", time.Now().Format(time.RFC3339))
	cluster, _ := CreateNewCluster(4)
	err := cluster.RemoveNode(cluster.Nodes[3].id) // Use the 'id' of the node to be removed
	if err != nil {
		t.Fatalf("Failed to remove node: %v", err)
	}
	if len(cluster.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(cluster.Nodes))
	}
	// Also, check that no remaining node has the removed node as a peer
	for _, node := range cluster.Nodes {
		for _, peer := range node.peers {
			if peer.term == 3 {
				t.Errorf("Removed node still referenced by peers")
			}
		}
	}
}

// TestLeaderElection simulates leader election by promoting one node and checking its state.
func TestLeaderElection(t *testing.T) {
	t.Logf("[%s] Starting TestLeaderElection", time.Now().Format(time.RFC3339))
	cluster, _ := CreateNewCluster(3)
	// Simulate leader election by manually setting a node as leader
	cluster.Nodes[0].state = "leader"

	leader := cluster.findLeader()
	if leader == nil || leader.state != "leader" {
		t.Errorf("Leader election failed, no leader found")
	}
}

// TestLogReplication verifies that commands are correctly appended to the leader's log.
func TestLogReplication(t *testing.T) {
	t.Logf("[%s] Starting TestLogReplication", time.Now().Format(time.RFC3339))
	cluster, _ := CreateNewCluster(3)
	cluster.Nodes[0].state = "leader"

	err := cluster.SetData("key1", 101)
	if err != nil {
		t.Errorf("Failed to set data on leader: %v", err)
	}

	leader := cluster.findLeader()
	if len(leader.log) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(leader.log))
	}

	if leader.log[0].Command != "key1=101" {
		t.Errorf("Log entry incorrect, expected 'key1=101', got '%v'", leader.log[0].Command)
	}
}

// TestFailover simulates leader failure and ensures a new leader is elected.
func TestFailover(t *testing.T) {
	t.Logf("[%s] Starting TestFailOver", time.Now().Format(time.RFC3339))
	cluster, _ := CreateNewCluster(5)
	cluster.Nodes[0].state = "leader"
	cluster.Nodes[1].state = "follower"
	cluster.Nodes[2].state = "follower"
	cluster.Nodes[3].state = "follower"
	cluster.Nodes[4].state = "follower"

	// Simulate leader node failure
	cluster.Nodes[0].state = "follower"

	// Expect a new leader to be elected from the remaining nodes
	newLeader := cluster.findLeader()
	if newLeader != nil && newLeader.state == "leader" {
		t.Errorf("Failover test failed, no new leader should be elected immediately")
	}

	// Simulate election and promote a new leader
	cluster.Nodes[1].state = "leader"
	newLeader = cluster.findLeader()
	if newLeader == nil || newLeader.state != "leader" {
		t.Errorf("Failover test failed, new leader was not elected")
	}
}
