// main.go

package dive

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	PingInterval time.Duration = time.Millisecond * 10
)

func randomMember(members map[string]bool) (member string) {
	index := rand.Intn(len(members))
	count := 0

	for member = range members {
		if count == index {
			break
		}
		count++
	}

	return
}

type Node struct {
	Members map[string]bool
	Joins   []string
	Id      int
}

func (n *Node) Address() string {
	return fmt.Sprintf("tmp/dive_%d.node", n.Id)
}

func (n *Node) heartbeet() {
	for {
		if len(n.Members) > 0 {
			other := randomMember(n.Members)
			n.Ping(other)
		}

		time.Sleep(PingInterval)
	}
}

func NewNode(seedAddress string, id int) *Node {
	node := &Node{
		Members: make(map[string]bool),
		Joins:   make([]string, 0),
		Id:      id,
	}
	if seedAddress != "" {
		node.Members[seedAddress] = true
	}
	go node.Serve()
	go node.heartbeet()
	return node
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
