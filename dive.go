// main.go

package dive

import (
	"fmt"
	"log"
	"math"
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
	Members       map[string]*LocalRecord
	Id            int
	alive         bool
	addMember     chan *BasicRecord
	updateMember  chan *BasicRecord
	failMember    chan *BasicRecord
	pingList      []*BasicRecord
	requestMember chan bool
	returnMember  chan BasicRecord
}

type BasicRecord struct {
	Status
	Address string
}

type LocalRecord struct {
	// Might pass around the count for more network traffic
	// and faster distribution, but probably not a good idea.
	// count   int
	BasicRecord
	SendCount int
}

func NewNodeRecord(address string) *LocalRecord {
	return &LocalRecord{BasicRecord: BasicRecord{Address: address}}
}

func (n *Node) NextPing(index int) *BasicRecord {
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

func (n *Node) PickMembers() []*BasicRecord {
	outMembers := make([]*BasicRecord, 0)
	for _, nodeRecord := range n.Members {
		if float64(nodeRecord.SendCount) > math.Log(float64(len(n.Members))) {
			log.Println("Not sending", nodeRecord, len(n.Members))
			continue
		}

		nodeRecord.SendCount++
		outMembers = append(outMembers, &nodeRecord.BasicRecord)
	}
	return outMembers
}

func (n *Node) heartbeat() {
	for {
		if n.alive && len(n.Members) > 0 {
			n.requestMember <- true
			other := <-n.returnMember
			go n.Ping(other)
		}

		time.Sleep(PingInterval)
	}
}

func (n *Node) keepMemberUpdated() {
	var basic *BasicRecord
	addr := n.Address()
	pingIndex := 0
	for {
		select {
		case basic = <-n.updateMember:
			node := new(LocalRecord)
			node.BasicRecord = basic
			node.SendCount = n.Members[basic.Address].SendCount

			if basic.Address != addr {
				n.Members[basic.Address] = node
			}
		case basic = <-n.failMember:
		case basic = <-n.addMember:
			if basic.Address != "" && basic.Address != addr {
				n.pingList = append(n.pingList, basic)
				i := len(n.pingList) - 1
				j := rand.Intn(i + 1)
				n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
				n.Members[basic.Address] = basic
			}
		case _ = <-n.requestMember:
			n.returnMember <- *n.NextPing(pingIndex)
			pingIndex++
		}
	}
}

func NewNode(seedAddress string) *Node {
	node := &Node{
		Members:       make(map[string]*LocalRecord),
		Id:            time.Now().Nanosecond(),
		alive:         true,
		addMember:     make(chan *BasicRecord, 1), // might need to be buffered?
		updateMember:  make(chan *BasicRecord, 1), // might need to be buffered?
		failMember:    make(chan *BasicRecord, 1), // might need to be buffered?
		pingList:      make([]*LocalRecord, 0),
		requestMember: make(chan bool, 1),
		returnMember:  make(chan BasicRecord, 1),
	}

	node.addMember <- NewNodeRecord(seedAddress)

	go node.keepMemberUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
