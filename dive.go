// dive.go

package dive

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

var (
	PingInterval time.Duration = time.Millisecond * 60
)

type Status int
type EventType int

func (s Status) String() string {
	if s == Alive {
		return "Alive"
	}
	return "Dead"
}

const (
	Alive Status = iota
	Suspected
	Failed
)

const (
	Join EventType = iota
	Fail
)

type Event struct {
	Kind     EventType
	Address  string
	MetaData interface{}
}

// The internal structure for a node
// holds and seperates channels, addresses,
// and current members. Useful for testing.
type Node struct {

	// A map from addresses to their LocalRecords
	Members map[string]*LocalRecord
	Host    string
	Port    int

	// Arbitrary information sent over gossip protocol - e.g. load information
	MetaData interface{}

	// Channels for thread-safety
	evalMember   chan *BasicRecord
	addMember    chan *BasicRecord
	updateMember chan *BasicRecord
	failMember   chan *BasicRecord
	// Exported chan for handling them outside.
	Events chan *Event

	// Used to request the next ping safely
	requestMember chan bool
	// Used to return the next ping safely
	returnMember chan BasicRecord

	// Used to call
	NextPing func() *BasicRecord

	// This is for round-robin pinging
	alive bool
}

func (n *Node) AddMember() chan *BasicRecord {
	return n.addMember
}

func GetAliveFromMap(records map[string]*LocalRecord) []*LocalRecord {
	output_list := make([]*LocalRecord, 0)
	for _, rec := range records {
		if rec.Status == Alive {
			output_list = append(output_list, rec)
		}
	}
	return output_list
}

// Record passed to other nodes
type BasicRecord struct {
	Status
	Address  string
	MetaData interface{}
}

func (basic *BasicRecord) toEvent(kind EventType) *Event {
	return &Event{kind, basic.Address, basic.MetaData}
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

func init() {
	// Run on as many cores as possible
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Be random each time
	rand.Seed(time.Now().UnixNano())
}

// Constructor for Local Record
func NewLocalRecord(address string) *LocalRecord {
	return &LocalRecord{BasicRecord: BasicRecord{Address: address}}
}

// copied from the interwebs
func shuffleLocal(a []*LocalRecord) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

// Return next record in round-robin list
// thread-safe
func (n *Node) setUpNextPing() func() *BasicRecord {
	arr := []*LocalRecord{}
	i := 0
	var f func() *BasicRecord
	f = func() *BasicRecord {
		if i == len(arr) {
			arr = GetAliveFromMap(n.Members)
			shuffleLocal(arr)
			i = 0
		}
		if len(arr) == 0 {
			fmt.Println("Why?")
		}

		out := arr[i]
		if out.Status == Failed {
			i++
			return f()
		}
		i++
		return &out.BasicRecord
	}
	return f
}

// Get the address of a node
func (n *Node) Address() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// Artificially kill a node
func (n *Node) Kill() {
	n.alive = false
}

// Artificially revive a node
func (n *Node) Revive() {
	n.alive = true
}

func (b *LocalRecord) isSendable() bool {
	return b.SendCount < 3
}

// Choose the records to send to other nodes
// Only takes ones that haven't been sent
// too many times before
func (n *Node) PickMembers() []*BasicRecord {
	outMembers := make([]*BasicRecord, 0)
	for _, nodeRecord := range n.Members {
		// if float64(nodeRecord.SendCount) > math.Log(float64(len(n.Members))) {
		if !nodeRecord.isSendable() {
			continue
		}

		nodeRecord.SendCount++
		outMembers = append(outMembers, &nodeRecord.BasicRecord)
	}

	// str := "["
	// for _, m := range outMembers {
	// 	str = str + " "
	// 	str = str + m.Address
	// 	str = str + " "
	// 	str = str + m.Status.String()
	// }
	// fmt.Println(str, "]")
	return outMembers
}

func LocalFromBasic(basic *BasicRecord) *LocalRecord {
	rec := new(LocalRecord)
	rec.BasicRecord = *basic
	return rec
}

// Start pinging every heartbeat
func (n *Node) heartbeat() {
	for {
		if n.alive && len(GetAliveFromMap(n.Members)) > 0 {
			n.requestMember <- true
			other := <-n.returnMember
			go n.Ping(other)
		}
		time.Sleep(PingInterval)
	}
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
			rec := LocalFromBasic(basic)
			rec.SendCount = 0
			n.Members[basic.Address] = rec
		}
	}
}

func (n *Node) handleAddMember(basic *BasicRecord) {
	addr := n.Address()
	if basic.Address != "" && basic.Address != addr {
		n.Members[basic.Address] = LocalFromBasic(basic)
		if n.Events != nil {
			n.Events <- basic.toEvent(Join)
		}
	}
}

func (n *Node) handleFailMember(basic *BasicRecord) {
	n.Members[basic.Address] = LocalFromBasic(basic)
	if n.Events != nil {
		n.Events <- basic.toEvent(Fail)
	}
}

func (n *Node) handleRequestMember(basic *BasicRecord) {
	n.returnMember <- *n.NextPing()
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
func NewNode(host string, port int, seed *BasicRecord, events chan *Event) *Node {
	node := &Node{
		Members:       make(map[string]*LocalRecord),
		Host:          host,
		Port:          port,
		alive:         true,
		evalMember:    make(chan *BasicRecord, 1), // buffering?
		addMember:     make(chan *BasicRecord, 1), // buffering?
		failMember:    make(chan *BasicRecord, 1),
		updateMember:  make(chan *BasicRecord, 1),
		requestMember: make(chan bool, 1),
		returnMember:  make(chan BasicRecord, 1),
		Events:        events,
	}

	node.addMember <- seed
	node.NextPing = node.setUpNextPing()

	go node.keepNodeUpdated()
	go node.Serve()
	go node.heartbeat()

	return node
}
