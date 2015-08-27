// dive_test.go

package dive

import (
	"fmt"
	"testing"
	"time"
)

func printDots() {
	fmt.Print(".")
	time.Sleep(60 * time.Millisecond)
	printDots()
}

func init() {
	fmt.Print("Testing")
	go printDots()
}

const (
	ClusterSize int           = 10
	Propagation time.Duration = time.Duration(4 * ClusterSize)
)

func checkNotPing(t *testing.T, nodes []*Node) {
	for _, node := range nodes {
		for _, member := range node.Members {
			if member.isSendable() {
				t.Errorf("Member %s still pingin: %s", node.Address(), member.Address)
			}

		}
		if len(node.PickMembers()) > 0 {
			t.Errorf("Member %s is sending", node.Address())
		}
	}
}

func checkMembers(t *testing.T, nodes []*Node) {
	for _, node := range nodes {
		for _, member := range node.Members {
			if member.Status != Alive {
				t.Errorf("Member failed: %s, %s", member.Address, node.Address())
			}
		}

		if len(node.Members) != len(nodes)-1 {
			t.Errorf("Node %s thinks there are %d nodes!", node.Address(), len(node.Members))
		}
	}
}

func checkFailure(t *testing.T, nodes []*Node, failed *Node) {
	for _, node := range nodes {
		for _, member := range node.Members {
			if member.Address == failed.Address() && member.Status != Failed {
				t.Errorf("%s thinks %s is alive", node.Address(), member.Address)
			}

			if member.Address != failed.Address() && member.Status != Alive {
				t.Errorf("%s thinks %s is dead", node.Address(), member.Address)
			}
		}
	}
}

var port int = 3000

func NewCluster(size int) []*Node {
	nodes := make([]*Node, ClusterSize)

	first := NewNode("localhost", port, &BasicRecord{Address: ""}, nil)
	port++
	nodes[0] = first

	time.Sleep(PingInterval)

	for i := 1; i < ClusterSize; i++ {
		nodes[i] = NewNode("localhost", port, &BasicRecord{Address: first.Address()}, nil)
		port++
	}

	return nodes
}

func printStatus(nodes []*Node) {
	for _, n := range nodes {
		fmt.Println("Alive: ", n.alive)
	}
}

func TestBasicJoin(t *testing.T) {
	nodes := NewCluster(ClusterSize)

	time.Sleep(PingInterval * Propagation)

	checkMembers(t, nodes)
}

func TestFailures(t *testing.T) {
	nodes := NewCluster(ClusterSize)
	time.Sleep(PingInterval * Propagation)
	checkMembers(t, nodes)

	failed := nodes[4]
	failed.Kill()
	time.Sleep(PingInterval * Propagation * 2)
	checkFailure(t, nodes, failed)
	checkNotPing(t, nodes)

	failed.Revive()
	time.Sleep(PingInterval * Propagation)
	checkMembers(t, nodes)
}

func TestStopPing(t *testing.T) {
	nodes := NewCluster(ClusterSize)

	time.Sleep(PingInterval * Propagation)

	checkNotPing(t, nodes)
}
