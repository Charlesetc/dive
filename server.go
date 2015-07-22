// Server.go

package dive

import (
	//	"fmt"
	"net"
	"net/rpc"
	"os"
)

// Server

type Server struct {
	node *Node
}

type Option struct {
	Address string
	Joins   []string
}

type Reply struct {
	Ack bool
}

func (n *Node) Serve() {
	rpcs := rpc.NewServer()
	s := &Server{node: n}
	rpcs.Register(s)

	os.Remove(n.Address())

	l, err := net.Listen("unix", n.Address())

	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go rpcs.ServeConn(conn)
	}

}

func (s *Server) Ping(o *Option, r *Reply) error {
	address := o.Address
	if address != s.node.Address() {

		for _, joinedNode := range o.Joins {
			if joinedNode != s.node.Address() {
				s.node.Members[joinedNode] = true
			}
		}

		_, existance := s.node.Members[address]

		if !existance {
			s.node.Joins = append(s.node.Joins, address)
			s.node.Members[address] = true
		}
	}
	r.Ack = true
	return nil
}

// Client

func (n *Node) Ping(address string) bool {
	conn, err := rpc.Dial("unix", address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	r := new(Reply)
	o := new(Option)
	o.Address = n.Address()
	o.Joins = n.Joins

	err = conn.Call("Server.Ping", o, r)
	return r.Ack
}
