// Server.go

package dive

import (
	"fmt"
	"net"
	"net/rpc"
	"time"
)

//// Server

type Server struct {
	node *Node
}

// The input to an rpc service.
// what server.ping takes
// what node.ping gives
type Option struct {
	Address  string
	Nodes    []*BasicRecord
	MetaData interface{}
}

// The ouput to an rpc service.
// what server.ping writes to
// what node.ping gets out of server.ping
type Reply struct {
	Ack   bool
	Nodes []*BasicRecord
}

// Repeatedly listen for connections
// which server.ping handles
func (n *Node) Serve() {
	rpcs := rpc.NewServer()
	s := &Server{node: n}
	rpcs.Register(s)

	l, err := net.Listen("tcp", n.Address())

	if err != nil {
		panic(err)
	}

	for {
		if n.alive {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}
			go rpcs.ServeConn(conn)
		} else {
			time.Sleep(time.Nanosecond) //  PingInterval)
		}
	}
}

// A Handler for any incoming connections.
// The server "is pinged"
func (s *Server) Ping(o *Option, r *Reply) error {
	address := o.Address

	for _, joinedNode := range o.Nodes {
		s.node.updateMember <- joinedNode
	}

	if _, exists := s.node.Members[address]; exists {
		// change status to alive.
		s.node.evalMember <- &BasicRecord{Address: address, MetaData: o.MetaData}
	} else {
		// add a new one.
		// this is continually being called.
		s.node.addMember <- &BasicRecord{Address: address, MetaData: o.MetaData}
	}

	r.Ack = true

	// PickMembers calls requestMember <-
	// so it is syncroneously after
	// the updateMember above.
	// so conflicting messages shouldn't
	// be an issue. Yay!
	r.Nodes = s.node.PickMembers()
	return nil
}

// Client

func dial(address string) (*rpc.Client, error) {
	var err error
	var conn *rpc.Client
	success := make(chan *rpc.Client)

	go func() {
		conn, err = rpc.Dial("tcp", address)

		for err != nil {
			conn, err = rpc.Dial("tcp", address)
			time.Sleep(PingInterval / 3)
		}

		success <- conn
	}()

	select {
	case conn := <-success:
		return conn, nil
	case <-time.After(PingInterval):
		return nil, err
	}
}

// Useful function for calling any method on
// a remote receiver
func call(address string, method string, o interface{}, r interface{}) chan bool {
	conn, err := dial(address)
	if err != nil {
		return nil // will wait until the timeout
	}
	resp := make(chan bool)

	go func() {
		err := conn.Call(method, o, r)
		conn.Close()

		if err != nil {
			fmt.Println("Error", err)
			return
		}

		resp <- true
	}()

	return resp
}

// Sends a message.
// The node "pings other"
func (n *Node) Ping(other BasicRecord) {
	address := other.Address
	r := new(Reply)
	o := new(Option)
	o.Address = n.Address()
	o.MetaData = n.MetaData
	o.Nodes = n.PickMembers()

	resp := call(address, "Server.Ping", o, r)

	// Either get a response or timeout
	select {
	case <-resp:
		for _, nodeRecord := range r.Nodes {
			n.updateMember <- nodeRecord
		}
	case <-time.After(PingInterval / 3):
		other.Status = Failed
		n.failMember <- &other
	}
}
