// main.go

package dive

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	ping_interval time.Duration = time.Millisecond * 100
)

type Node struct {
	Members map[string]bool
	Id      int
}

func (n *Node) Address() string {
	return fmt.Sprintf("tmp/dive_%d.node", n.Id)
}

func (n *Node) heartbeet() {
	for {
		desired_value := rand.Intn(len(n.Members))
		var other string
		i := 0
		for other = range n.Members {
			if i == desired_value {
				break
			}
			i++
		}
		n.Ping(other)
		fmt.Println(other)
		time.Sleep(ping_interval)
	}
}

func NewNode(seed_address string, id int) *Node {
	node := &Node{Members: make(map[string]bool), Id: id}
	if seed_address != "" {
		fmt.Println(seed_address)
		node.Members[seed_address] = true
	}
	go node.Serve()
	go node.heartbeet()
	return node
}
