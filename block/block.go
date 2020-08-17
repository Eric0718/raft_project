package block

import (
	"bytes"
	"encoding/json"
	"fmt"

	"kto/types"
	"kto/until/miscellaneous"

	"kto/transaction"

	"golang.org/x/crypto/sha3"
)

func NewBlock() *Block {
	b := &Block{

		Txres: make(map[types.Address]uint64),
		
	}

	return b
}

func (b *Blocks) Tx() []*transaction.Transaction {
	return b.Txs
}
func (b *Block) Tx() []*transaction.Transaction {
	return b.Txs
}

func (b *Blocks) ToBytes() []byte {
	block_byte, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return block_byte
}
func (b *Block) ToBytes() []byte {
	block_byte, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return block_byte
}

func (b *Block) FromBytes(data []byte) error {
	return json.Unmarshal(data, &b)
}

func (block *Block) SetHash() {

	heightBytes := miscellaneous.E64func(block.Height)

	trans, _ := json.Marshal(block.Txs)
	b := miscellaneous.E64func(uint64(block.Timestamp))
	blockBytes := bytes.Join([][]byte{heightBytes, block.PrevBlockHash, trans, b}, []byte{})
	// 4. 生成Hash
	hash := sha3.Sum256(blockBytes)
	block.Hash = hash[:]

}

func (block *Blocks) SetHash() {

	heightBytes := miscellaneous.E64func(block.Height)

	trans, _ := json.Marshal(block.Txs)
	b := miscellaneous.E64func(uint64(block.Timestamp))
	blockBytes := bytes.Join([][]byte{heightBytes, block.PrevBlockHash, trans, b}, []byte{})
	// 4. 生成Hash
	hash := sha3.Sum256(blockBytes)
	block.Hash = hash[:]

}

// func NewBlockResult() *block.BlockResult {
// 	tx_list := &block.BlockResult{

// 		Txres: make(map[types.Address]uint64),
// 	}
// 	return tx_list
// }
