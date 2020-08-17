package txpool

import (
	"container/heap"
	"crypto/sha256"
	"fmt"
	"kto/block"
	"kto/blockchain"
	"kto/contract/exec"
	"kto/contract/parser"
	"kto/transaction"
	"kto/types"
	"kto/until/logger"
	"kto/until/merkle"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ed25519"
)

const Totalquantity = 500

type TxHeap []*transaction.Transaction

func (h TxHeap) Len() int { return len(h) }

func (h TxHeap) Less(i, j int) bool {

	if h[i].From == h[j].From {
		if h[i].GetNonce() == h[j].GetNonce() {
			return h[i].GetTime() < h[j].GetTime()
		}
		return h[i].GetNonce() < h[j].GetNonce()
	}

	return h[i].GetTime() < h[j].GetTime()
}

func (h TxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]

}

func (h *TxHeap) Push(x interface{}) {
	*h = append(*h, x.(*transaction.Transaction))
}

func (h *TxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h *TxHeap) Get(i int) interface{} {
	old := *h
	n := len(old)
	x := old[n-i]
	return x
}

func (h *TxHeap) check(fromAddr types.Address, nonce uint64) bool {
	var count = 0
	for _, tx := range *h {
		if fromAddr == tx.From {
			if nonce == tx.Nonce {
				return false
			}
			count++
		}
	}
	if count >= 50 {
		return false
	}
	return true
}

type Txpool struct {
	List    *TxHeap
	queue   *TxHeap
	Pending []*transaction.Transaction
	item    map[types.Address]uint64
	balance map[types.Address]int64
	mu      sync.RWMutex
}

func readinfo() []byte {

	vp := viper.New()

	vp.SetConfigName("nodecfg")
	vp.AddConfigPath("./conf")
	vp.SetConfigType("yaml")
	err := vp.ReadInConfig()
	if err != nil {
		fmt.Printf("config file error: %s\n", err)
		os.Exit(1)
	}
	str := vp.GetString("qtj")

	pub := types.AddrTopub(str)
	if len(pub) != 32 {
		fmt.Println("len(pub) != 32 ===========")
		return []byte("")
	}
	return pub
}

func (p *Txpool) readcfginfo() {

	vp := viper.New()

	vp.SetConfigName("nodecfg")
	vp.AddConfigPath("./conf")
	vp.SetConfigType("yaml")
	err := vp.ReadInConfig()
	if err != nil {
		fmt.Printf("config file error: %s\n", err)
		os.Exit(1)
	}

}
func New() *Txpool {
	pool := &Txpool{
		List:    new(TxHeap),
		queue:   new(TxHeap),
		item:    make(map[types.Address]uint64),
		balance: make(map[types.Address]int64),
	}
	heap.Init(pool.List)
	heap.Init(pool.queue)

	return pool
}

func (p *Txpool) SetTxpollToNil() {

	var txlist []*transaction.Transaction
	*p.List = TxHeap(txlist)

	fmt.Println("txpool list len =", p.List.Len())
}
func (p *Txpool) Add(tx *transaction.Transaction, Bc *blockchain.Blockchain) error {

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.List.Len() > 1500 {
		return errtxoutrange
	}
	ok := verify(*tx, Bc)
	if !ok {
		return errtx
	}

	if ok := p.List.check(tx.From, tx.Nonce); !ok {
		return errtomuch
	}
	tx.Time = time.Now().Unix()
	heap.Push(p.List, tx)
	return nil
}

func (p *Txpool) Pendings(Bc *blockchain.Blockchain) {

	p.mu.Lock()
	defer p.mu.Unlock()
	var tx *transaction.Transaction
	fmt.Println("p.List len = ", p.List.Len())
	for {

		if len(p.Pending) >= Totalquantity {
			break
		}
		if p.List.Len() == 0 {
			break
		}
		tx = heap.Pop(p.List).(*transaction.Transaction)
		n, err := Bc.GetNonce(tx.From.AddressToByte())
		if err != nil {
			logger.Infof("Pendings GetNonce is err = %v", err)
		}

		if tx.Nonce < n {
			continue
		}
		if _, ok := p.balance[tx.From]; !ok {
			fromba, _ := Bc.GetBalance(tx.From.AddressToByte())

			num, err1 := Bc.GetLockBalance(string(tx.From.AddressToByte()))
			if err1 != nil {
				num = 0
			}
			fromba -= num //可用余额
			if fromba <= 0 {
				logger.Infof("fromba <= 0 ")
				continue
			}
			if fromba < tx.Amount {
				logger.Infof("fromba < tx.Amount")
				continue
			}
			p.balance[tx.From] = int64(fromba)
		}

		if tx.Nonce == n {
			//符合
			if _, ok := p.item[tx.From]; !ok {
				//如果没有，表示是第一次，进行赋值

				if p.balance[tx.From] >= int64(tx.Amount) {
					p.item[tx.From] = n
					if tx.Script != "" {
						cdb := Bc.Getcdb()
						sc := parser.Parser([]byte(tx.Script))
						logger.Infof("tx.Script = %s,sc = %v, from = %s", tx.Script, sc, string(tx.From.AddressToByte()))
						e, err := exec.New(cdb, sc, string(tx.From.AddressToByte()))
						if err != nil {
							tx.Errmsg = "exec New is failed "
							//logger.Infof("tx.Script = %s,sc = %v, from = %s", tx.Script, sc, string(tx.From.AddressToByte()))
							continue
						}
						tx.Root = e.Root()
						p.balance[tx.From] -= int64(tx.Fee)
					}
					p.Pending = append(p.Pending, tx)
					p.balance[tx.From] -= int64(tx.Amount)
					continue
				}
				continue

			} else {

				//如果p.item[tx.From]有 判断是否和现在的相同，如果相同代表重复
				continue

			}
		} else {
			//不符合还要判断是否和map的值是否符合，不符合就放进新的队列
			if _, ok := p.item[tx.From]; !ok {
				//如果有 判断是否和现在的相同，如果相同代表重复
				p.item[tx.From] = n
				heap.Push(p.queue, tx)
				continue
			}

			if tx.Nonce == p.item[tx.From]+1 {
				//符合map的值

				if p.balance[tx.From] >= int64(tx.Amount) {
					p.item[tx.From] = tx.Nonce
					if tx.Script != "" {
						cdb := Bc.Getcdb()
						sc := parser.Parser([]byte(tx.Script)) //script := fmt.Sprintf("transfer \"%s\" %d \"%s\"", in.Symbol, At, in.To)
						e, err := exec.New(cdb, sc, string(tx.From.AddressToByte()))
						if err != nil {
							tx.Errmsg = "exec New is failed "
							continue
						}
						tx.Root = e.Root()
						p.balance[tx.From] -= int64(tx.Fee)

					}
					p.Pending = append(p.Pending, tx)
					p.balance[tx.From] -= int64(tx.Amount)
					continue
				}
				continue
			}
			heap.Push(p.queue, tx)
		}
	}

	if p.queue.Len() != 0 {
		var txs []*transaction.Transaction
		for i := 0; i < p.queue.Len(); i++ {
			tx := heap.Pop(p.queue).(*transaction.Transaction)
			txs = append(txs, tx)
		}
		for j := 0; j < len(txs); j++ {
			tm := time.Now().UTC().Unix()
			if tm-txs[j].Time < 10 {
				heap.Push(p.List, txs[j])
			}
		}

	}

	logger.Infof("pening len =%d ", len(p.Pending))
	logger.Infof("p.List len = %d", p.List.Len())
	fmt.Println("p.pening len = ", len(p.Pending))
	fmt.Println("p.List len = ", p.List.Len())

	for k, _ := range p.balance {
		delete(p.balance, k)
	}
	for k, _ := range p.item {
		delete(p.item, k)
	}
	return
}

func (p *Txpool) GetPending() []*transaction.Transaction {
	return p.Pending
}

func (p *Txpool) GetPendingLen() int {
	return len(p.Pending)
}

func verify(tx transaction.Transaction, Bc *blockchain.Blockchain) bool {
	ktoaddr := tx.From.AddressToByte()
	if string(ktoaddr)[:3] != "Kto" {
		logger.Infof("string(ktoaddr)[:3] != kto===========")
		return false
	}
	pub := tx.From.AddrToPub()
	if len(pub) != 32 {
		logger.Infof("len(pub) != 32 ===========")
		return false
	}

	toaddr := tx.To.AddressToByte()
	if string(toaddr)[:3] != "Kto" {
		logger.Infof("string(toaddr)[:3] != kto===========")
		return false
	}
	topub := tx.To.AddrToPub()
	if len(topub) != 32 {
		logger.Infof("len(topub) != 32 ===========")
		return false
	}

	ok := ed25519.Verify(ed25519.PublicKey(pub), tx.Hash, tx.Signature)
	if ok != true {
		logger.Infof("ed25519 Verify failed,please check public  ===========")
		return false
	}

	nonce, err := Bc.GetNonce(tx.From.AddressToByte())
	if err != nil {
		if err1 := Bc.SetNonce(tx.From.AddressToByte(), uint64(1)); err1 != nil {
			logger.Infof("setnonce is err:%v", err1)
			return false
		}
		nonce = 1
	}

	if tx.Nonce < nonce {
		logger.Infof("tx.BlockNumber = %d,tx.Nonce err,  tx.Nonce=%d,nocne = %d，addr= %s", tx.BlockNumber, tx.Nonce, nonce, string(tx.From.AddressToByte()))
		return false
	}
	bal, err := Bc.GetBalance(tx.From.AddressToByte())
	if err != nil {
		logger.Infof("balance is err:%v", err)

		return false
	}
	lb, _ := Bc.GetLockBalance(string(tx.From.AddressToByte()))

	ulb := bal - lb
	if ulb < tx.Amount {
		logger.Infof("unlock ulb %v < tx.Amount %v,lock balance = %v,addr %s bal = %v.", ulb, tx.Amount, lb, string(tx.From.AddressToByte()), bal)
		return false
	}

	if tx.Script != "" {
		if tx.Fee < 500000 || tx.Fee > ulb {
			logger.Infof("token Fee is err %v,%v,please check", tx.Fee, ulb)
			return false
		}
	}

	if tx.Amount < 500000 || tx.Amount > bal {
		logger.Infof("tx.Amount < 500000 || tx.Amount > bal")
		return false
	}

	if tx.Ord.Signature != nil && len(tx.Ord.Signature) != 0 {
		pb := readinfo()
		ok = ed25519.Verify(ed25519.PublicKey(pb), tx.Ord.Hash, tx.Ord.Signature)
		if ok != true {
			logger.Infof("tx.Ord.Signature is bad,please check")
			return false
		}
	}

	return true
}

func VerifyBlcok(b block.Blocks, Bc *blockchain.Blockchain) bool {

	var trans [][]byte
	for _, tx := range b.Txs {
		trans = append(trans, tx.Tobytes())
	}
	if trans != nil {
		tree := merkle.New(sha256.New(), trans)
		if ok := tree.VerifyNode(b.Root); ok {
			logger.Infof("tree.VerifyNode error")
			return false
		}

		for _, tx := range b.Txs {
			if !verify(*tx, Bc) {
				logger.Infof("verify error")
				return false
			}
		}

	}

	return true
}

func (p *Txpool) Filter(b block.Blocks) {
	logger.Infof("Start Filter\n")
	p.mu.Lock()
	defer p.mu.Unlock()

	txlist := []*transaction.Transaction(*p.List)

	for _, tx := range b.Txs {

		for index := 0; index < len(txlist); index++ {
			if tx.From == txlist[index].From && tx.Nonce == txlist[index].Nonce {
				if len(txlist) > index {
					txlist = append(txlist[:index], txlist[index+1:]...)
					index -= 1
				}

			}
		}
	}
	*p.List = TxHeap(txlist)

	fmt.Println("Filter after is p.List len = ", p.List.Len())

	if p.List.Len() != 0 {
		var txs []*transaction.Transaction
		for i := 0; i < p.List.Len(); i++ {
			tx := heap.Pop(p.List).(*transaction.Transaction)
			txs = append(txs, tx)
		}
		for j := 0; j < len(txs); j++ {
			tm := time.Now().UTC().Unix()
			if tm-txs[j].Time < 10 {
				heap.Push(p.List, txs[j])
			}
		}

	}
	logger.Infof("End Filter\n")

}

// func (p *Txpool) Pendings(Bc *blockchain.Blockchain) {

// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	var tx *transaction.Transaction
// 	fmt.Println("p.List len = ", p.List.Len())
// 	for {

// 		//logger.Infof("p.List len = %d", p.List.Len())

// 		if len(p.Pending) >= Totalquantity {
// 			break
// 		}
// 		if p.List.Len() == 0 {
// 			break
// 		}
// 		tx = heap.Pop(p.List).(*transaction.Transaction)
// 		n, err := Bc.GetNonce(tx.From.AddressToByte())
// 		if err != nil {
// 			logger.Infof("Pendings GetNonce is err = %v", err)
// 		}

// 		// {
// 		// 	fromba, _ := tx.Get(Tx.From.AddressToByte())
// 		// 	num, err := tx.Mget([]byte("lock"), Tx.From.AddressToByte())
// 		// 	if err != nil {

// 		// 		num = miscellaneous.E64func(0)

// 		// 	}
// 		// 	bal, _ := miscellaneous.D64func(num)
// 		// 	fbal, _ := miscellaneous.D64func(fromba)
// 		// 	fbal -= bal

// 		// 	balance[Tx.From] = int64(fbal)
// 		// }

// 		if tx.Script != "" {
// 			cdb := Bc.Getcdb()
// 			sc := parser.Parser([]byte(tx.Script))
// 			e, err := exec.New(cdb, sc, string(tx.From.AddressToByte()))
// 			if err != nil {
// 				tx.Errmsg = "exec New is failed "
// 			}
// 			tx.Root = string(e.Root())
// 		}

// 		if tx.Nonce == n {
// 			//符合
// 			if nc, ok := p.item[tx.From]; !ok {
// 				//如果没有，表示是第一次，进行赋值
// 				p.item[tx.From] = n
// 				p.Pending = append(p.Pending, tx)
// 				continue
// 			} else {
// 				if nc == tx.Nonce {
// 					//如果p.item[tx.From]有 判断是否和现在的相同，如果相同代表重复
// 					continue
// 				}
// 				p.item[tx.From] = n
// 				p.Pending = append(p.Pending, tx)
// 				continue
// 			}
// 		} else {
// 			//不符合还要判断是否和map的值是否符合，不符合就放进新的队列
// 			if tx.Nonce == p.item[tx.From] {
// 				//如果有 判断是否和现在的相同，如果相同代表重复
// 				continue
// 			}
// 			//ne := p.item[tx.From] + 1

// 			if tx.Nonce == p.item[tx.From] + 1 {
// 				//符合map的值
// 				p.item[tx.From] = tx.Nonce
// 				p.Pending = append(p.Pending, tx)
// 				continue
// 			}
// 			heap.Push(p.queue, tx)

// 		}

// 	}

// 	// for {
// 	// 	if p.queue.Len() != 0 {
// 	// 		tx := heap.Pop(p.queue).(*transaction.Transaction)
// 	// 		tm := time.Now().UTC().Unix()
// 	// 		if tm-tx.Time > 1000 {

// 	// 			continue
// 	// 		}
// 	// 		heap.Push(p.List, tx)
// 	// 		continue
// 	// 	}
// 	// 	break
// 	// }
// 	if p.queue.Len() != 0 {
// 		var txs []*transaction.Transaction
// 		for i := 0; i < p.queue.Len(); i++ {
// 			tx := heap.Pop(p.queue).(*transaction.Transaction)
// 			txs = append(txs, tx)
// 		}
// 		for j := 0; j < len(txs); j++ {
// 			tm := time.Now().UTC().Unix()
// 			if tm-txs[j].Time < 30 {
// 				heap.Push(p.List, txs[j])
// 			}
// 		}

// 	}

// 	logger.Infof("pening len =%d ", len(p.Pending))
// 	logger.Infof("p.List len = %d", p.List.Len())
// 	fmt.Println("p.pening len = ", len(p.Pending))
// 	fmt.Println("p.List len = ", p.List.Len())

// 	return
// }
