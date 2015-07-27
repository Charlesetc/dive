// dive_test.go

package dive

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func printDots() {
	fmt.Print(".")
	time.Sleep(300 * time.Millisecond)
	printDots()
}

func init() {
	exec.Command("rm", "-r", "tmp").Output()
	exec.Command("mkdir", "-p", "tmp").Output()
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
			t.Errorf("Node %d thinks there are %d node(s)!", node.Id, len(node.Members))
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

func destroyCluster(nodes []*Node) {
	for _, node := range nodes {
		os.Remove(node.Address())
	}
}

func NewCluster(size int) []*Node {
	nodes := make([]*Node, ClusterSize)

	first := NewNode("")
	nodes[0] = first
	seed := first.Address()

	time.Sleep(PingInterval)

	for i := 1; i < ClusterSize; i++ {
		nodes[i] = NewNode(seed)
	}

	return nodes
}

// func TestAddMember(t *testing.T) {
// 	node1 := NewNode("")
// 	node2 := NewNode("")
//
// }

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
	// printStatus(nodes)
	checkMembers(t, nodes)
}

func TestReJoin(t *testing.T) {
	nodes := NewCluster(ClusterSize)
	time.Sleep(PingInterval * Propagation)
	checkMembers(t, nodes)

	node := NewNode(nodes[2].Address())
	nodes = append(nodes, node)

	time.Sleep(PingInterval * Propagation)
	checkMembers(t, nodes)
}

func TestStopPing(t *testing.T) {
	nodes := NewCluster(ClusterSize)

	time.Sleep(PingInterval * Propagation)

	checkNotPing(t, nodes)
}

//c TestTwoClusters(t *testing.T) {
//odes1 := NewCluster(ClusterSize)
//odes2 := NewCluster(ClusterSize)
//odes1[0]
//
