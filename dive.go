// main.go

package dive

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

const (
	PingInterval time.Duration = time.Millisecond * 10
)

type Status int

const (
	ALIVE Status = iota
	SUSPECT
	FAIL
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())
}

func randomMember(members map[string]*NodeRecord) (member string) {
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
	Members   map[string]*NodeRecord
	Id        int
	alive     bool
	addMember chan *NodeRecord
	pingList  []*NodeRecord
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
		for i := range pingList {
			j := rand.Intn(i + 1)
			n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
		}
	}

	return n.pingList[index]
}

func (n *Node) Address() string {
	return fmt.Sprintf("/var/tmp/dive_%d.node", n.Id)
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
			other := randomMember(n.Members)
			n.Ping(other)
		}

		time.Sleep(PingInterval)
	}
}

func (n *Node) keepMemberUpdated() {
	var nodeRecord *NodeRecord
	addr := n.Address()
	for {
		nodeRecord = <-n.addMember
		if nodeRecord.Address != "" && nodeRecord.Address != addr {
			n.Members[nodeRecord.Address] = nodeRecord
			n.pingList = append(n.pingList, nodeRecord)
			i := len(n.pingList) - 1
			j := rand.Intn(i + 1)
			n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
		}
	}
}

func NewNode(seedAddress string, id int) *Node {
	node := &Node{
		Members:   make(map[string]*NodeRecord),
		Id:        id,
		alive:     true,
		addMember: make(chan *NodeRecord, 1), // might need to be buffered?
		pingList:  make([]*NodeRecord),
	}

	node.addMember <- &NodeRecord{Address: seedAddress}

	go node.keepMemberUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
