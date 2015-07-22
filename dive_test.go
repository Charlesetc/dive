// dive_test.go

package dive

import (
	"fmt"
	"testing"
	"time"
	// "time"
)

const (
	numberOfNodes int = 10
)

func checkMembers(t *testing.T, nodes []*Node) {
	for _, node := range nodes {
		if len(node.Members) != len(nodes)-1 {
			t.Errorf("Node %d thinks there are %d node(s)!", node.Id, len(node.Members))
		}
	}
}

func TestDive(t *testing.T) {
	nodes := make([]*Node, numberOfNodes)

	first := NewNode("", 0)
	nodes[0] = first

	seed := first.Address()

	for i := 1; i < numberOfNodes; i++ {
		nodes[i] = NewNode(seed, i)
	}

	fmt.Println("Waiting...")
	time.Sleep(PingInterval * 50)
	fmt.Println("Done waiting.")

	checkMembers(t, nodes)
}
