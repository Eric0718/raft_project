package node

import (
	"encoding/gob"
	"kto/blockchain"
	"kto/p2p"
	"kto/transaction"
	"kto/txpool"
	"kto/types"
)

type Node interface {
	Run()
	Stop()
	Join([]string) error
	Broadcast(*transaction.Transaction)
}

type node struct {
	p    p2p.P2P
	pool *txpool.Txpool
	bc   *blockchain.Blockchain
}

func init() {
	gob.Register(types.Address{})
	gob.Register(transaction.Order{})
	gob.Register(transaction.Transaction{})
}

func New(port int, name string, address string, pool *txpool.Txpool, bc *blockchain.Blockchain) (*node, error) {
	n := &node{pool: pool, bc: bc}
	p, err := p2p.New(p2p.Config{port, name, address}, n, recv)
	if err != nil {
		return nil, err
	}
	n.p = p
	return n, nil
}

func (n *node) Broadcast(tx *transaction.Transaction) {
	data, _ := p2p.Encode(*tx)
	n.p.Broadcast(data)
}

func (n *node) Run() {
	n.p.Run()
}

func (n *node) Stop() {
	n.p.Stop()
}

func (n *node) Join(ns []string) error {
	return n.p.Join(ns)
}

func recv(u interface{}, data []byte) {
	var tx transaction.Transaction

	n := u.(*node)
	if err := p2p.Decode(data, &tx); err == nil {
		n.pool.Add(&tx, n.bc)
	}
}
