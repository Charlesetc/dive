// main.go

package dive

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"
)

const (
	PingInterval time.Duration = time.Millisecond * 10
	Timeout                    = PingInterval / 3
)

type Status int

const (
	Alive Status = iota
	Suspected
	Failed
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())
}

type Node struct {
	Members       map[string]*NodeRecord
	Id            int
	alive         bool
	addMember     chan *NodeRecord
	pingList      []*NodeRecord
	requestMember chan bool
	returnMember  chan *NodeRecord
}

type NodeRecord struct {
	// Might pass around the count for more network traffic
	// and faster distribution, but probably not a good idea.
	// count   int
	Status
	Address string
}

func (n *Node) NextPing(index int) *NodeRecord {
	index = index % len(n.pingList)

	if index == 0 {
		for i := range n.pingList {
			j := rand.Intn(i + 1)
			n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
		}
	}

	return n.pingList[index]
}

func (n *Node) Address() string {
	return fmt.Sprintf("tmp/dive_%d.node", n.Id)
}

func (n *Node) Kill() {
	n.alive = false
}

func (n *Node) Revive() {
	n.alive = true
}

func (n *Node) PickMembers() []*NodeRecord {
	outMembers := make([]*NodeRecord, 0)
	for _, nodeRecord := range n.Members {
		outMembers = append(outMembers, nodeRecord)
	}
	return outMembers
}

func (n *Node) heartbeat() {
	for {
		if n.alive && len(n.Members) > 0 {
			n.requestMember <- true
			other := <-n.returnMember
			go n.Ping(other.Address)
		}

		time.Sleep(PingInterval)
	}
}

func (n *Node) keepMemberUpdated() {
	var nodeRecord *NodeRecord
	addr := n.Address()
	pingIndex := 0
	for {
		select {
		case nodeRecord = <-n.addMember:
			if nodeRecord.Address != "" && nodeRecord.Address != addr {
				if _, ok := n.Members[nodeRecord.Address]; !ok {
					n.pingList = append(n.pingList, nodeRecord)
					i := len(n.pingList) - 1
					j := rand.Intn(i + 1)
					n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
				}
				log.Println("Member update", n.Address(), nodeRecord)

				n.Members[nodeRecord.Address] = nodeRecord
			}
		case _ = <-n.requestMember:
			n.returnMember <- n.NextPing(pingIndex)
			pingIndex++
		}
	}
}

func NewNode(seedAddress string) *Node {
	node := &Node{
		Members:       make(map[string]*NodeRecord),
		Id:            time.Now().Nanosecond(),
		alive:         true,
		addMember:     make(chan *NodeRecord, 1), // might need to be buffered?
		pingList:      make([]*NodeRecord, 0),
		requestMember: make(chan bool, 1),
		returnMember:  make(chan *NodeRecord, 1),
	}

	node.addMember <- &NodeRecord{Address: seedAddress}

	go node.keepMemberUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
