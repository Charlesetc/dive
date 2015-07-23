// Server.go

package dive

import (
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
	Ack   bool
	Joins []string
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

	for _, joinedNode := range o.Joins {
		s.node.addMember <- joinedNode
	}

	if _, ok := s.node.Members[address]; !ok {
		s.node.addJoin <- address
		s.node.addMember <- address
	}

	r.Ack = true
	r.Joins = s.node.Joins
	return nil
}

// Client

func (n *Node) Ping(address string) bool {
	// fmt.Println(address)
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

	if err != nil {
		panic(err)
	}

	for _, joinedNode := range r.Joins {
		n.addMember <- joinedNode
	}

	return r.Ack
}
