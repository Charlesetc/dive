// dive_test.go

package dive

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func init() {
	exec.Command("rm", "-r", "tmp").Output()
	exec.Command("mkdir", "-p", "tmp").Output()
}

const (
	ClusterSize int           = 10
	Propagation time.Duration = time.Duration(2 * ClusterSize)
)

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

	time.Sleep(PingInterval * Propagation)

	checkFailure(t, nodes, failed)

	failed.Revive()

	time.Sleep(PingInterval * Propagation)

	checkMembers(t, nodes)
}
