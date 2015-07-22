// dive_test.go

package dive

import (
	"fmt"
	"testing"
	"time"
)

const (
	ClusterSize int = 10
)

func checkMembers(t *testing.T, nodes []*Node) {
	for _, node := range nodes {
		if len(node.Members) != len(nodes)-1 {
			t.Errorf("Node %d thinks there are %d node(s)!", node.Id, len(node.Members))
		}
	}
}

func NewCluster(size int) []*Node {
	nodes := make([]*Node, ClusterSize)

	first := NewNode("", 0)
	nodes[0] = first
	seed := first.Address()

	for i := 1; i < ClusterSize; i++ {
		nodes[i] = NewNode(seed, i)
	}

	return nodes
}

func TestDive(t *testing.T) {
	nodes := NewCluster(ClusterSize)

	fmt.Println("Waiting...")
	time.Sleep(PingInterval * 50)

	checkMembers(t, nodes)
}
