// Server.go

package dive

import (
	"fmt"
	"net"
	"net/rpc"
	"time"
)

// Server

type Server struct {
	node *Node
}

type Option struct {
	Address string
	Nodes   []*BasicRecord
}

type Reply struct {
	Ack   bool
	Nodes []*BasicRecord
}

func (n *Node) Serve() {
	rpcs := rpc.NewServer()
	s := &Server{node: n}
	rpcs.Register(s)

	l, err := net.Listen("unix", n.Address())

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
			time.Sleep(time.Millisecond * PingInterval)
		}
	}
}

func (s *Server) Ping(o *Option, r *Reply) error {
	address := o.Address

	for _, joinedNode := range o.Nodes {
		s.node.addMember <- joinedNode
	}

	if _, exists := s.node.Members[address]; exists {
		s.node.evalMember <- &BasicRecord{Address: address}
	} else {
		s.node.addMember <- &BasicRecord{Address: address}
	}

	r.Ack = true
	r.Nodes = s.node.PickMembers()
	return nil
}

// Client

func dial(address string) *rpc.Client {
	conn, err := rpc.Dial("unix", address)

	if err != nil {
		panic(err)
	}

	return conn
}

func call(address string, method string, o interface{}, r interface{}) chan bool {
	conn := dial(address)
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

func (n *Node) Ping(other BasicRecord) {
	address := other.Address
	r := new(Reply)
	o := new(Option)
	o.Address = n.Address()
	o.Nodes = n.PickMembers()

	resp := call(address, "Server.Ping", o, r)

	select {
	case <-resp:
		for _, nodeRecord := range r.Nodes {
			n.updateMember <- nodeRecord
		}
		n.evalMember <- &other
	case <-time.After(Timeout):
		other.Status = Failed
		n.failMember <- &other
	}
}
