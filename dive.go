// main.go

package dive

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
)

const (
	PingInterval time.Duration = time.Millisecond * 100
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
	evalMember    chan *BasicRecord
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

func NewLocalRecord(address string) *LocalRecord {
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
			continue
		}

		nodeRecord.SendCount++
		outMembers = append(outMembers, &nodeRecord.BasicRecord)
	}

	number_of_alive := 0
	number_of_failed := 0

	for _, mem := range n.Members {
		if mem.Status == Alive {
			number_of_alive++
		} else {
			fmt.Println(mem.Address)
			number_of_failed++
		}
	}
	fmt.Println("--\t", number_of_alive, "\t", number_of_failed, "\t", n.Address())
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
		case basic = <-n.evalMember:
			if basic.Status != n.Members[basic.Address].Status {
				n.Members[basic.Address].Status = basic.Status
				n.Members[basic.Address].SendCount = 0
			}
		case basic = <-n.updateMember:
			if _, exists := n.Members[basic.Address]; exists {
				if basic.Address != addr {
					n.Members[basic.Address].Status = basic.Status
				} // Otherwise : raise error or tell someone
				break
			} // Else does not exist:
			rec := new(LocalRecord)
			rec.BasicRecord = *basic
			rec.SendCount = -1
			n.Members[basic.Address] = rec
		case basic = <-n.addMember:
			if basic.Address != "" && basic.Address != addr {

				// flip recent one
				n.pingList = append(n.pingList, basic)
				i := len(n.pingList) - 1
				j := rand.Intn(i + 1)
				n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]

				rec := new(LocalRecord)
				rec.BasicRecord = *basic
				rec.SendCount = 0
				n.Members[basic.Address] = rec
			}
		case basic = <-n.failMember:
			rec := new(LocalRecord)
			rec.BasicRecord = *basic
			rec.SendCount = 0
			n.Members[basic.Address] = rec
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
		evalMember:    make(chan *BasicRecord, 1), // buffering?
		addMember:     make(chan *BasicRecord, 1), // buffering?
		failMember:    make(chan *BasicRecord, 1),
		updateMember:  make(chan *BasicRecord, 1),
		pingList:      make([]*BasicRecord, 0),
		requestMember: make(chan bool, 1),
		returnMember:  make(chan BasicRecord, 1),
	}

	node.addMember <- &BasicRecord{Address: seedAddress}

	go node.keepMemberUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
