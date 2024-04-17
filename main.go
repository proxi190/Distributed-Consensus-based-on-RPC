package main

import (
	"DisSys/raft"
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	var cluster *raft.RaftCluster
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Distributed Raft Consensus based on RPC")
	fmt.Println("========================================")
	fmt.Println("Enter command:")

	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		tokens := strings.Fields(input)

		if len(tokens) == 0 {
			fmt.Println("Empty input, please try again.")
			continue
		}

		command, err := strconv.Atoi(tokens[0])
		if err != nil {
			fmt.Println("Invalid command, please try again.")
			continue
		}

		switch command {
		case 1: // Create cluster
			if len(tokens) < 2 {
				fmt.Println("Not enough arguments to create cluster.")
				break
			}
			peers, err := strconv.Atoi(tokens[1])
			if err != nil {
				fmt.Println("Invalid peer number.")
				break
			}
			cluster, err = raft.CreateNewCluster(peers)
			if err != nil {
				fmt.Printf("Error when creating cluster: %v\n", err)
			} else {
				fmt.Printf("Cluster with %d peers created.\n", peers)
			}
		case 2: // Set data
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				break
			}
			if len(tokens) < 3 {
				fmt.Println("Not enough arguments to set data.")
				break
			}
			key := tokens[1]
			value, err := strconv.Atoi(tokens[2])
			if err != nil {
				fmt.Println("Invalid value for key.")
				break
			}
			err = cluster.SetData(key, value)
			if err != nil {
				fmt.Printf("Error when setting data: %v\n", err)
			} else {
				fmt.Printf("Data set: %s = %d\n", key, value)
			}
		case 3: // Get data
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				break
			}
			if len(tokens) < 2 {
				fmt.Println("Not enough arguments to get data.")
				break
			}
			key := tokens[1]
			value, err := cluster.GetData(key)
			if err != nil {
				fmt.Printf("Error when getting data: %v\n", err)
			} else {
				fmt.Printf("Data: %s = %d\n", key, value)
			}
		case 4: // Add node
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				break
			}
			if len(tokens) < 2 {
				fmt.Println("Not enough arguments to add node.")
				break
			}
			newNodeID, err := strconv.Atoi(tokens[1])
			if err != nil {
				fmt.Println("Invalid node ID.")
				break
			}
			err = cluster.AddNode(newNodeID)
			if err != nil {
				fmt.Printf("Error adding node: %v\n", err)
			} else {
				fmt.Println("Node added successfully.")
			}
		case 5: // Remove node
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				break
			}
			if len(tokens) < 2 {
				fmt.Println("Not enough arguments to remove node.")
				break
			}
			nodeID, err := strconv.Atoi(tokens[1])
			if err != nil {
				fmt.Println("Invalid node ID.")
				break
			}
			err = cluster.RemoveNode(nodeID)
			if err != nil {
				fmt.Printf("Error removing node: %v\n", err)
			} else {
				fmt.Println("Node removed successfully.")
			}
		case 6: // Exit
			fmt.Println("Exiting.")
			return
		case 7: // Trigger manual leader election
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				continue
			}
			if len(tokens) < 2 {
				fmt.Println("Not enough arguments to trigger election.")
				continue
			}
			nodeID, err := strconv.Atoi(tokens[1]) // Need to pass the node ID to trigger the election
			if err != nil || nodeID < 0 || nodeID >= len(cluster.Nodes) {
				fmt.Println("Invalid node ID.")
				continue
			}
			cluster.Nodes[nodeID].PerformElectionTimeout()
			fmt.Printf("Election timeout manually triggered for Node %d.\n", nodeID)
		case 8: // Print cluster status
			if cluster == nil {
				fmt.Println("No cluster initialized.")
				continue
			}
			cluster.PrintStatus()
		default:
			fmt.Println("Unknown command, please try again.")
		}
	}
}
