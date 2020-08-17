package block

import (
	"kto/transaction"
	"kto/types"
)

type Block struct {
	Height        uint64                     `json:"height"`        //当前块号
	PrevBlockHash []byte                     `json:"prevBlockHash"` //上一块的hash
	Txs           []*transaction.Transaction `json:"txs"`           //交易数据
	Root          []byte                     `json:"root"`          //默克根
	Version       uint64                     `json:"version"`       //版本号
	Timestamp     int64                      `json:"timestamp"`     //时间戳
	Hash          []byte                     `json:"hash"`          //当前块hash
	Miner         []byte                     `json:"miner"`
	Txres         map[types.Address]uint64   `json:"txres"`
	FirstTx       []MinerTx                  `json:"firstTx"`
}

type MinerTx struct {
	Amount     uint64 `json:"amount"`
	RecAddress []byte `json:"recaddress"`
}

type Blocks struct {
	Height        uint64                     `json:"height"`        //当前块号
	PrevBlockHash []byte                     `json:"prevBlockHash"` //上一块的hash
	Txs           []*transaction.Transaction `json:"txs"`           //交易数据
	Root          []byte                     `json:"root"`          //默克根
	Version       uint64                     `json:"version"`       //版本号
	Timestamp     int64                      `json:"timestamp"`     //时间戳
	Hash          []byte                     `json:"hash"`          //当前块hash
	Miner         []byte                     `json:"miner"`
	Res           []Block_Res                `json:"res"`
	FirstTx       []MinerTx                  `json:"firstTx"`
}

type Block_Res struct {
	Address types.Address
	Balance uint64
}
