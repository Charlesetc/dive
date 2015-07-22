// main.go

package dive

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	ping_interval time.Duration = time.Millisecond * 300
)

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
		if len(n.Members) == 0 {
			time.Sleep(ping_interval)
			continue
		}
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
		time.Sleep(ping_interval)
	}
}

func NewNode(seed_address string, id int) *Node {
	node := &Node{
		Members: make(map[string]bool),
		Joins:   make([]string, 0),
		Id:      id,
	}
	if seed_address != "" {
		node.Members[seed_address] = true
	}
	go node.Serve()
	go node.heartbeet()
	return node
}

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}
