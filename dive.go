// main.go

package dive

import (
	"fmt"
	"math/rand"
	// "runtime"
	"time"
)

const (
	PingInterval time.Duration = time.Millisecond * 10
)

func init() {
	// runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())
}

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
	Members   map[string]bool
	Joins     []string
	Id        int
	addJoin   chan string
	addMember chan string
}

func (n *Node) Address() string {
	return fmt.Sprintf("tmp/dive_%d.node", n.Id)
}

func (n *Node) heartbeat() {
	for {
		if len(n.Members) > 0 {
			other := randomMember(n.Members)
			n.Ping(other)
		}

		time.Sleep(PingInterval)
	}
}

func (n *Node) keepJoinUpdated() {
	for {
		var s string
		s = <-n.addJoin
		n.Joins = append(n.Joins, s)
	}
}

func (n *Node) keepMemberUpdated() {
	var s string
	addr := n.Address()
	for {
		s = <-n.addMember
		if s != "" && s != addr {
			n.Members[s] = true
		}
	}
}

func NewNode(seedAddress string, id int) *Node {
	node := &Node{
		Members:   make(map[string]bool),
		Joins:     make([]string, 0),
		Id:        id,
		addJoin:   make(chan string, 10),
		addMember: make(chan string, 10),
	}

	node.addMember <- seedAddress

	go node.keepJoinUpdated()
	go node.keepMemberUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
