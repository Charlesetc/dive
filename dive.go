// dive.go

package dive

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
)

const (
	// Time between Pings
	PingInterval time.Duration = time.Millisecond * 60
	// Time it takes for a ping to fail
	Timeout = PingInterval / 3
)

type Status int

const (
	Alive Status = iota
	Suspected
	Failed
)

func init() {
	// Run on as many cores as possible
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Be random each time
	rand.Seed(time.Now().UnixNano())
}

// The internal structure for a node
// holds and seperates channels, addresses,
// and current members. Useful for testing.
type Node struct {

	// A map from addresses to their LocalRecords
	Members map[string]*LocalRecord
	// Id used to generate address, as of now.
	Id int

	// Channels for thread-safety
	evalMember   chan *BasicRecord
	addMember    chan *BasicRecord
	updateMember chan *BasicRecord
	failMember   chan *BasicRecord

	// Used to request the next ping safely
	requestMember chan bool
	// Used to return the next ping safely
	returnMember chan BasicRecord

	// This is for round-robin pinging
	pingList  []*BasicRecord
	pingIndex int
	alive     bool
}

// Record passed to other nodes
type BasicRecord struct {
	Status
	Address string
}

// Record kept locally
// keeps # of times sent.
type LocalRecord struct {
	// Might pass around the count for more network traffic
	// and faster distribution, but probably not a good idea.
	// count   int
	BasicRecord
	SendCount int
}

// Constructor for Local Record
func NewLocalRecord(address string) *LocalRecord {
	return &LocalRecord{BasicRecord: BasicRecord{Address: address}}
}

// Return next record in round-robin list
// thread-safe
func (n *Node) NextPing() *BasicRecord {
	index := n.pingIndex
	index = index % len(n.pingList)

	if index == 0 {
		for i := range n.pingList {
			j := rand.Intn(i + 1)
			n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
		}
	}

	return n.pingList[index]
}

// Get the address of a node
func (n *Node) Address() string {
	return fmt.Sprintf("tmp/dive_%d.node", n.Id)
}

// Artificially kill a node
func (n *Node) Kill() {
	n.alive = false
}

// Artificially revive a node
func (n *Node) Revive() {
	n.alive = true
}

// Choose the records to send to other nodes
// Only takes ones that haven't been sent
// too many times before
func (n *Node) PickMembers() []*BasicRecord {
	outMembers := make([]*BasicRecord, 0)
	for _, nodeRecord := range n.Members {
		if float64(nodeRecord.SendCount) > math.Log(float64(len(n.Members))) {
			continue
		}

		nodeRecord.SendCount++
		outMembers = append(outMembers, &nodeRecord.BasicRecord)
	}

	return outMembers
}

// Start pinging every heartbeat
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

func (n *Node) addToPingList(basic *BasicRecord) {
	n.pingList = append(n.pingList, basic)
	i := len(n.pingList) - 1
	j := rand.Intn(i + 1)
	n.pingList[i], n.pingList[j] = n.pingList[j], n.pingList[i]
}

func (n *Node) handleEvalMember(basic *BasicRecord) {
	addr := n.Address()
	if basic.Address != addr {
		if basic.Status != n.Members[basic.Address].Status {
			n.Members[basic.Address].Status = basic.Status
			n.Members[basic.Address].SendCount = 0
		}
	}
}

func (n *Node) handleUpdateMember(basic *BasicRecord) {
	addr := n.Address()
	if basic.Address != addr {
		if _, exists := n.Members[basic.Address]; exists {
			if basic.Address != addr {
				n.Members[basic.Address].Status = basic.Status
			}
		} else {
			n.addToPingList(basic)
			rec := new(LocalRecord)
			rec.BasicRecord = *basic
			rec.SendCount = -1
			n.Members[basic.Address] = rec
		}
	}
}

func (n *Node) handleAddMember(basic *BasicRecord) {
	addr := n.Address()
	if basic.Address != "" && basic.Address != addr {
		n.addToPingList(basic)

		rec := new(LocalRecord)
		rec.BasicRecord = *basic
		rec.SendCount = 0
		n.Members[basic.Address] = rec
	}
}

func (n *Node) handleFailMember(basic *BasicRecord) {
	rec := new(LocalRecord)
	rec.BasicRecord = *basic
	rec.SendCount = 0
	n.Members[basic.Address] = rec
}

func (n *Node) handleRequestMember(basic *BasicRecord) {
	n.returnMember <- *n.NextPing()
	n.pingIndex++
}

// Look at Node's channels and process
// incoming requests.
func (n *Node) keepNodeUpdated() {
	var basic *BasicRecord
	for {
		select {
		case basic = <-n.evalMember:
			n.handleEvalMember(basic)
		case basic = <-n.updateMember:
			n.handleUpdateMember(basic)
		case basic = <-n.addMember:
			n.handleAddMember(basic)
		case basic = <-n.failMember:
			n.handleFailMember(basic)
		case _ = <-n.requestMember:
			n.handleRequestMember(basic)
		}
	}
}

// Make a New Node
// if seedAddress is empty,
// it's the seed node and the address
// is ignored
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

	go node.keepNodeUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
