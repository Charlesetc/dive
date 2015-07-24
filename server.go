// Server.go

package dive

import (
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

func call(conn *rpc.Client, method string, args interface{}, reply interface{}) error {
	err := conn.Call("Server.Ping", args, reply)
	return err
}

func (n *Node) Ping(address string) bool {
	conn := dial(address)
	defer conn.Close()

	r := new(Reply)
	o := new(Option)
	o.Address = n.Address()
	o.Nodes = n.PickMembers()

	err := call(conn, "Server.Ping", o, r)

	if err != nil {
		panic(err)
	}

	for _, nodeRecord := range r.Nodes {
		n.addMember <- nodeRecord
	}

	return r.Ack
}
