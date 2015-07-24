// Server.go

package dive

import (
	"log"
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
	Nodes   []*NodeRecord
}

type Reply struct {
	Ack   bool
	Nodes []*NodeRecord
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

	s.node.addMember <- &NodeRecord{Address: address}

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
		err := conn.Call("Server.Ping", o, r)
		conn.Close()

		if err != nil {
			log.Println("Error", err)
		}

		resp <- true
	}()

	return resp
}

func (n *Node) Ping(address string) bool {
	r := new(Reply)
	o := new(Option)
	o.Address = n.Address()
	o.Nodes = n.PickMembers()

	resp := call(address, "Server.Ping", o, r)

	select {
	case <-resp:
		for _, nodeRecord := range r.Nodes {
			n.addMember <- nodeRecord
		}

		return true
	case <-time.After(Timeout):
		log.Println("Failed", n.Address(), address)
		return false
	}
}
