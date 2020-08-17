package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"kto/block"
	"kto/contract/exec"
	"kto/contract/parser"
	"kto/transaction"
	"kto/types"
	"kto/until/logger"
	"kto/until/merkle"
	"kto/until/miscellaneous"
	"kto/until/store"
	"kto/until/store/bg"
	"sync"
	"time"
)

type Blockchains interface {
	AddBlock(*block.Block) error
	NewBlock([]transaction.Transaction) (*block.Block, error)

	GetNonce([]byte) (uint64, error)
	SetNonce([]byte, uint64) error
	GetBalance([]byte) (uint64, error)
	Height() (uint64, error)
	GetBlockByHash([]byte) (*block.Block, error)
	GetBlockByHeight(uint64) (*block.Block, error)

	GetTransaction([]byte) (*transaction.Transaction, error)
	GetTransactionHeight([]byte) (uint64, error)
	GetTransactionByAddr([]byte) (error, []transaction.Transaction)
	GetTransactionByHash([]byte) (error, transaction.Transaction)
	GetMaxBlockHeight([]byte) (uint64, error)
}

type Blockchain struct {
	db  store.DB
	cdb store.DB
	mu  sync.RWMutex
	// logFile     string
	// logSaveDays int
	// logLevel    int
	// logSaveMode int
	// logFileSize int64
}

func New() *Blockchain {
	logger.Infof("new blockchain")
	bgs := bg.New("blockchain.db")
	bgc := bg.New("contract.db")
	Bc := &Blockchain{db: bgs, cdb: bgc}

	return Bc
}

func (Bc *Blockchain) Getcdb() store.DB {
	return Bc.cdb
}

func (Bc *Blockchain) GetMaxBlockHeight() (uint64, error) {
	num, err := Bc.db.Get([]byte("height"))
	if err != nil {
		logger.Infof("GetMaxBlockHeight in :%v", err)
		return 0, err
	}
	bal, err := miscellaneous.D64func(num)
	if err != nil {
		return 0, err
	}
	return bal, err
}

func (Bc *Blockchain) GetBalance(addr []byte) (uint64, error) {

	Bc.mu.RLock()
	defer Bc.mu.RUnlock()

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	num, err := tx.Get(addr)
	if err != nil {
		logger.Infof("GetBalance in :%v", err)
		return 0, err
	}
	bal, err := miscellaneous.D64func(num)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return bal, err
}

func (Bc *Blockchain) GetTokenBalance(addr, symbol []byte) (uint64, error) {

	Bc.mu.RLock()
	defer Bc.mu.RUnlock()

	b, err := exec.Balance(Bc.cdb, string(symbol), string(addr)) // 2，代比， 3，地址
	if err != nil {
		logger.Infof("symbol :%v ,addr:%v", string(symbol), string(addr))
		return 0, errors.New("err:please check token symbol")
	}
	return b, nil
}

func (Bc *Blockchain) NewBlock(tr []*transaction.Transaction, minaddr, Ds, Cm, QTJ []byte) (*block.Block, error) {
	//创建新区
	Bc.mu.Lock()
	defer Bc.mu.Unlock()

	logger.Infof("Start new a block!\n")
	// h_c, err := Bc.Height()
	// if err != nil {
	// 	logger.Infof("Bc.Height() err !\n")
	// 	return nil, err
	// }
	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	h, err := tx.Get([]byte("height"))
	if err != nil {
		logger.Infof("Height Get in :%v", err)
		return nil, err
	}
	h_c, err := miscellaneous.D64func(h)
	if err != nil {
		logger.Infof("%v", err)
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	var trans [][]byte
	for _, txc := range tr {
		txc.BlockNumber = h_c + 1
		trans = append(trans, txc.Tobytes())
	}

	tree := merkle.New(sha256.New(), trans)

	root := tree.GetMtHash()

	ht := miscellaneous.E64func(h_c)
	bhash, err := Bc.db.Mget(ht, []byte("hash"))
	if err != nil {
		logger.Infof("Bc.db.Mget bhash err !\n")
		return nil, err
	}
	v1, err := miscellaneous.D64func(ht)
	if err != nil {
		logger.Infof("miscellaneous.D64func(ht) err !\n")
		return nil, err
	}
	hd := v1 + 1
	timetamp := time.Now().Unix()

	txres := make(map[types.Address]uint64)

	mintx := Distr(tr, minaddr, Ds, Cm, QTJ, hd)

	var newBlock block.Block = block.Block{hd, bhash, tr, root, 1, timetamp, nil, minaddr, txres, mintx}

	newBlock.SetHash()

	logger.Infof("Finish new a block! length of mintx = %v\n", len(mintx))

	return &newBlock, nil
}

func setMinerAccount(tx store.Transaction, to []byte, amount uint64) {
	tobalance, err := tx.Get(to)
	if err != nil {
		err1 := set_balance(tx, to, miscellaneous.E64func(0))
		if err1 != nil {
			logger.Infof("setMinerAccount is err: %v", err1)
		}
		tobalance = miscellaneous.E64func(0)
	}
	ToBalance, _ := miscellaneous.D64func(tobalance)
	ToBalance += amount
	Tobytes := miscellaneous.E64func(ToBalance)
	err2 := set_balance(tx, to, Tobytes)
	if err2 != nil {
		logger.Infof("setMinerAccount is err: %v", err2)
	}
}

func setMinerFee(tx store.Transaction, to []byte, amount uint64) {
	tobalance, err := tx.Get(to)
	if err != nil {
		err1 := set_balance(tx, to, miscellaneous.E64func(0))
		if err1 != nil {
			logger.Infof("setMinerAccount is err: %v", err1)
		}
		tobalance = miscellaneous.E64func(0)
	}
	ToBalance, _ := miscellaneous.D64func(tobalance)
	ToBalance += amount
	Tobytes := miscellaneous.E64func(ToBalance)
	err2 := set_balance(tx, to, Tobytes)
	if err2 != nil {
		logger.Infof("setMinerAccount is err: %v", err2)
	}
}

func fittle(tr []*transaction.Transaction, tx store.Transaction) []*transaction.Transaction {
	var addrAmount map[types.Address]int64
	addrAmount = make(map[types.Address]int64)

	for _, Tx := range tr {

		if _, ok := addrAmount[Tx.To]; !ok {
			tobalance, err := tx.Get(Tx.To.AddressToByte())
			if err != nil {

				tobalance = miscellaneous.E64func(0)
			}
			num, err := tx.Mget([]byte("lock"), Tx.To.AddressToByte())
			if err != nil {

				num = miscellaneous.E64func(0)

			}
			bal, _ := miscellaneous.D64func(num)
			tbal, _ := miscellaneous.D64func(tobalance)
			tbal -= bal

			addrAmount[Tx.To] = int64(tbal)

		}

		if _, ok := addrAmount[Tx.From]; !ok {
			fromba, _ := tx.Get(Tx.From.AddressToByte())
			num, err := tx.Mget([]byte("lock"), Tx.From.AddressToByte())
			if err != nil {

				num = miscellaneous.E64func(0)

			}
			bal, _ := miscellaneous.D64func(num)
			fbal, _ := miscellaneous.D64func(fromba)
			fbal -= bal

			addrAmount[Tx.From] = int64(fbal)
		}

	}

	var tx_s []*transaction.Transaction
	for _, Tx := range tr {

		fb := addrAmount[Tx.From] - int64(Tx.Amount)
		addrAmount[Tx.From] = fb

		if addrAmount[Tx.From] >= 0 {

			tx_s = append(tx_s, Tx)
		}

	}

	//b.Txs = tx_s

	return tx_s

}

func (Bc *Blockchain) AddBlock(b *block.Blocks, minaddr []byte) error {

	Bc.mu.Lock()
	defer Bc.mu.Unlock()

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()
	//拿出块高
	v, err := tx.Get([]byte("height"))
	if err != nil {
		logger.Infof("AddBlock get in :%v", err)
		return err
	}
	//byte转64   块高
	h, _ := miscellaneous.D64func(v)
	// if h-b.Height != 1 {
	// 	logger.Infof("h %v - b.Height %v != 1 :%v", h, b.Height, err)
	// 	panic("h  - b.Height  != 1 ")
	// }
	h++
	hash := b.Hash
	//高度->哈希

	err = tx.Mset(miscellaneous.E64func(h), []byte("hash"), hash)
	if err != nil {
		logger.Infof("AddBlock mset in :", err)
		return err
	}

	bt := b.ToBytes()
	//哈希-> 块
	err = tx.Set(hash, bt)
	if err != nil {
		logger.Infof("AddBlock setblock in :%v", err)
		return err
	}
	//高度对下个高度
	err = tx.Set(v, miscellaneous.E64func(h+1))
	if err != nil {
		logger.Infof("AddBlock setheight in :%v", err)
		return err
	}
	tx.Del([]byte("height"))
	tx.Set([]byte("height"), miscellaneous.E64func(h))
	txs := b.Tx()
	// logger.Infof("addblock===", len(b.FirstTx))
	for _, mintx := range b.FirstTx {
		//fmt.Println("range b.FirstTx======== addr = ", string(mintx.RecAddress), "AMount =", mintx.Amount)
		setMinerAccount(tx, mintx.RecAddress, mintx.Amount)

	}

	for _, Tx := range txs {

		if Tx.Script != "" {
			sc := parser.Parser([]byte(Tx.Script))
			e, err := exec.New(Bc.cdb, sc, string(Tx.From.AddressToByte()))
			if err != nil {
				Tx.Errmsg = "exec New is failed "
			}
			err = e.Flush()
			if err != nil {
				Tx.Errmsg = "flush code is failed"
			}
			setMinerFee(tx, minaddr, Tx.Fee)
		}
		err := setTxbyaddr(tx, Tx.From.AddressToByte(), *Tx)
		if err != nil {
			logger.Infof("AddBlock setTxbyaddr from in :%v", err)
			return err
		}
		err = setTxbyaddr(tx, Tx.To.AddressToByte(), *Tx)
		if err != nil {
			logger.Infof("AddBlock setTxbyaddr to in :%v", err)
			return err
		}

		setAccount(tx, Tx.From.AddressToByte(), Tx.To.AddressToByte(), Tx.Amount, Tx.Nonce, Tx)
		tx_data, err := json.Marshal(Tx)
		if err != nil {
			return err
		}
		//块hash->交易hash->交易数据
		setBlockdata(tx, b.Hash, Tx.Hash, tx_data, miscellaneous.E64func(h))

	}

	return tx.Commit()
}

func Distr(Txs []*transaction.Transaction, minaddr, Ds, Cm, QTJ []byte, height uint64) []block.MinerTx {
	var lenOrd, distr uint64 = 0, 0
	var minerTx []block.MinerTx
	var mtx block.MinerTx
	var total uint64 = 49460000000
	QTJ = types.AddrTopub(string(QTJ))
	x := height / 31536000
	for i := 0; uint64(i) <= x; i++ {

		if i == 0 {
			distr = total
			continue
		}

		distr = total * 8 / 10
		total = distr
	}

	for _, tr := range Txs {
		//if !reflect.ValueOf(tr.Ord).IsValid() {
		if tr.Ord.Signature != nil && len(tr.Ord.Signature) != 0 {
			if ok := ed25519.Verify(ed25519.PublicKey(QTJ), tr.Ord.Hash, tr.Ord.Signature); ok {
				lenOrd += 1
			}
		}
	}

	if lenOrd != 0 {

		fAmonut := distr / 10 / lenOrd
		for _, txOrd := range Txs {
			//if !reflect.ValueOf(txOrd.Ord).IsValid() {
			if txOrd.Ord.Signature != nil && len(txOrd.Ord.Signature) != 0 {
				if ok := ed25519.Verify(ed25519.PublicKey(QTJ), txOrd.Ord.Hash, txOrd.Ord.Signature); ok {

					mtx.Amount = fAmonut
					mtx.RecAddress = txOrd.Ord.Address.AddressToByte()
					minerTx = append(minerTx, mtx)
				}
			}
		}
		dsAmount := distr / 10 //10% 电商
		mtx.Amount = dsAmount
		mtx.RecAddress = Ds
		minerTx = append(minerTx, mtx)

	} else {

		dsAmount := distr * 2 / 10 //20% 电商
		mtx.Amount = dsAmount
		mtx.RecAddress = Ds
		minerTx = append(minerTx, mtx)
	}
	jsAmount := distr * 4 / 10 //40% 技术
	mtx.Amount = jsAmount
	mtx.RecAddress = minaddr
	minerTx = append(minerTx, mtx)

	sqAmount := distr * 4 / 10 //30% 社区
	mtx.Amount = sqAmount
	mtx.RecAddress = Cm
	minerTx = append(minerTx, mtx)

	return minerTx
}

func setTxbyaddr(tx store.Transaction, addr []byte, txs transaction.Transaction) error {

	// tx := db.NewTransaction()
	// defer tx.Cancel()

	tx_byte, err := json.Marshal(txs)
	if err != nil {
		logger.Infof("setTxbyaddr  in :%v", err)
		return err
	}
	err = tx.Mset(addr, txs.Hash, tx_byte)
	if err != nil {
		logger.Infof("setTxbyaddr Mset in :%v", err)
		return err
	}
	return err

}

func (Bc *Blockchain) GetNonce(n []byte) (uint64, error) {
	//获取nonce
	Bc.mu.RLock()
	defer Bc.mu.RUnlock()
	tx := Bc.db.NewTransaction()
	defer tx.Cancel()
	nonce, err := tx.Mget([]byte("nonce"), n)
	if err != nil {
		//logger.Infof("getnonce err = %v", err)
		Bc.SetNonce(n, uint64(1))
		return 1, nil
	}
	nu, err1 := miscellaneous.D64func(nonce)
	if err1 != nil {
		logger.Infof("D64func err = %v", err1)
		return 0, err1
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return nu, nil
}

func (Bc *Blockchain) Height() (uint64, error) {
	//获取当前高度
	Bc.mu.RLock()
	defer Bc.mu.RUnlock()
	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	h, err := tx.Get([]byte("height"))
	if err != nil {
		logger.Infof("Height Get in :%v", err)
		return 0, err
	}
	ht, err := miscellaneous.D64func(h)
	if err != nil {
		logger.Infof("%v", err)
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return ht, err
}

func (Bc *Blockchain) GetBlockByHash(hash []byte) (*block.Block, error) { //
	//获取区块

	b, err := Bc.db.Get(hash)
	if err != nil {
		logger.Infof("GetBlockByHash Get in :%v", err)
		return nil, err
	}
	var blo *block.Block = &block.Block{}
	json.Unmarshal(b, &blo)
	return blo, err

}

func (Bc *Blockchain) GetBlockByHeight(h uint64) (*block.Block, error) {
	//通过高度获取区块		//高度dui哈希-.kuai
	hash, err := Bc.db.Mget(miscellaneous.E64func(h), []byte("hash"))
	if err != nil {
		logger.Infof("1111,%v", err)
		return nil, err
	}
	B, err := Bc.db.Get(hash)
	if err != nil {
		logger.Infof("222,%v", err)
		return nil, err
	}
	logger.Infof("block size = ", len(B))
	blcok := &block.Block{}
	err = json.Unmarshal(B, blcok)
	if err != nil {
		logger.Infof("333,%v", err)
		return nil, err
	}
	return blcok, err
}

func (Bc *Blockchain) GetTransaction(hash []byte) (*transaction.Transaction, error) {
	//获取hash的交易
	tx, err := Bc.db.Get(hash)
	if err != nil {
		logger.Infof("GetTransaction Get in :%v", err)
		return nil, err
	}
	var tran *transaction.Transaction
	err = json.Unmarshal(tx, &tran)
	if err != nil {
		logger.Infof("%v", err)
		return nil, err
	}
	return tran, err
}

func (Bc *Blockchain) GetTransactionHeight(hash []byte) (uint64, error) {
	//获取交易在哪个高度块里
	tx_data, err := Bc.db.Get(hash)
	if err != nil {
		logger.Infof("GetTransaction Get in :%v", err)
		return 0, err
	}
	h, err := Bc.db.Mget(hash, tx_data)
	if err != nil {
		logger.Infof("GetTransaction Mget in :%v", err)
		return 0, err
	}
	height, err := miscellaneous.D64func(h)
	if err != nil {
		logger.Infof("%v", err)
		return 0, err
	}
	return height, nil
}

func set_nonce(tx store.Transaction, addr, nonce []byte) error {
	// tx := db.NewTransaction()
	// defer tx.Cancel()

	err := tx.Mset([]byte("nonce"), addr, nonce)

	if err != nil {
		logger.Infof("set_nonce Mset in :%v", err)
		return err
	}
	return err

}
func (Bc *Blockchain) SetNonce(addr []byte, n uint64) error {
	//设置nonce

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	nu := miscellaneous.E64func(n)
	err := tx.Mset([]byte("nonce"), addr, nu)
	if err != nil {
		logger.Infof("SetNonce Mset in :%v", err)
		return err
	}
	return tx.Commit()
}
func setBlockdata(tx store.Transaction, blockhash, txhash, txdata, h []byte) error {
	// tx := db.NewTransaction()
	// defer tx.Cancel()
	//块hash->交易hash->交易数据

	err := tx.Mset(blockhash, txhash, txdata)
	if err != nil {

		logger.Infof("setBlockdata Mset in :%v", err)
		return err
	}
	//交易hash->交易数据->块高
	err = tx.Mset(txhash, txdata, h)
	if err != nil {

		logger.Infof("setBlockdata Mset in :%v", err)
		return err
	}
	//交易hash->交易数据
	err = tx.Set(txhash, txdata)
	if err != nil {
		logger.Infof("setBlockdata Set in :%v", err)
		return err
	}
	return err
}

func setAccount(tx store.Transaction, from, to []byte, amount, nonce uint64, Tx *transaction.Transaction) {
	tobalance, err := tx.Get(to)
	if err != nil {
		set_balance(tx, to, miscellaneous.E64func(0))
		tobalance = miscellaneous.E64func(0)
	}
	fromba, _ := tx.Get(from)
	FromBalance, _ := miscellaneous.D64func(fromba)
	ToBalance, _ := miscellaneous.D64func(tobalance)
	if Tx.Fee != 0 {
		FromBalance -= amount
		FromBalance -= Tx.Fee
	} else {
		FromBalance -= amount
	}

	ToBalance += amount

	Frombytes := miscellaneous.E64func(FromBalance)
	Tobytes := miscellaneous.E64func(ToBalance)

	set_balance(tx, from, Frombytes)
	set_balance(tx, to, Tobytes)

	nonce += 1
	tx.Mdel([]byte("nonce"), from)
	set_nonce(tx, from, miscellaneous.E64func(nonce))

}

func set_balance(tx store.Transaction, addr, balance []byte) error {
	tx.Del(addr)
	err := tx.Set(addr, balance)
	if err != nil {
		logger.Infof("set_balance Set in :%v", err)
		return err
	}
	return err

}
func MinerAccount(tx store.Transaction, addr []byte, amount uint64) {
	tobalance, err := tx.Get(addr)
	if err != nil {
		set_balance(tx, addr, miscellaneous.E64func(0))
		tobalance = miscellaneous.E64func(0)
	}
	ToBalance, _ := miscellaneous.D64func(tobalance)
	ToBalance += amount
	Tobytes := miscellaneous.E64func(ToBalance)

	set_balance(tx, addr, Tobytes)
}
func (Bc *Blockchain) GetTransactionByAddr(addr []byte) (error, []transaction.Transaction) {
	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	_, txs_v, err := tx.Mkvs(addr)
	if err != nil {
		logger.Infof("mkvs err != nil ,%v", err)
		return err, nil
	}

	var trans []transaction.Transaction
	for _, tx := range txs_v {
		var tran transaction.Transaction
		err := json.Unmarshal(tx, &tran)
		if err != nil {
			logger.Infof("1234,%v", err)
			return err, nil
		}
		trans = append(trans, tran)
	}

	return tx.Commit(), trans
}
func (Bc *Blockchain) GetTransactionByHash(hash []byte) (*transaction.Transaction, error) {

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()
	var trans transaction.Transaction
	tx_byte, err := tx.Get(hash)
	if err != nil {
		logger.Infof("GetTransactionByHash Get in :%v", err)
		return nil, err
	}
	err = json.Unmarshal(tx_byte, &trans)
	if err != nil {
		logger.Infof("%v", err)
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		logger.Infof("%v", err)
		return nil, err
	}

	return &trans, nil

}

func (Bc *Blockchain) GetLockBalance(addr string) (uint64, error) {

	Bc.mu.RLock()
	defer Bc.mu.RUnlock()

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	num, err := tx.Mget([]byte("lock"), []byte(addr))
	if err != nil {
		//logger.Infof("GetLockBalance MGet in :%v", err)
		return 0, err
	}
	bal, err := miscellaneous.D64func(num)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		logger.Infof("%v", err)
		return 0, err
	}
	return bal, err
}

func (Bc *Blockchain) SetLockBalance(tx store.Transaction, addr string, amount uint64) error {

	Bc.mu.Lock()
	defer Bc.mu.Unlock()

	bal := miscellaneous.E64func(amount)

	_, err := tx.Mget([]byte("lock"), []byte(addr))
	if err == nil {
		err = tx.Mdel([]byte("lock"), []byte(addr))
		if err != nil {
			logger.Infof("SetLockBalance Mdel in :%v", err)
			return err
		}
		err = tx.Mset([]byte("lock"), []byte(addr), bal)
		if err != nil {
			logger.Infof("SetLockBalance Mset in :%v", err)
			return err
		}
		return err
	}

	err = tx.Mset([]byte("lock"), []byte(addr), bal)
	if err != nil {
		logger.Infof("SetLockBalance lock Mset in :%v", err)
		return err
	}

	return err
}

func (Bc *Blockchain) LockBalance(addr string, amount uint64) bool {

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	addrBal, _ := Bc.GetBalance([]byte(addr))

	addrLockbal, _ := Bc.GetLockBalance(addr)

	lb := addrLockbal + amount
	num := addrBal - addrLockbal
	if lb > addrBal || amount > addrBal || amount > uint64(num) {
		logger.Infof("please check amount")
		return false
	}

	amount += addrLockbal
	err := Bc.SetLockBalance(tx, addr, amount)
	if err != nil {
		return false
	}
	err = tx.Commit()
	if err != nil {
		logger.Infof("%v", err)
		return false
	}
	return true
}

func (Bc *Blockchain) Unlockbalance(addr string, amount uint64) bool {

	tx := Bc.db.NewTransaction()
	defer tx.Cancel()

	addrLockbal, _ := Bc.GetLockBalance(addr)
	if amount > addrLockbal {
		logger.Infof("please check amount")
		return false
	}
	at := addrLockbal - amount
	err := Bc.SetLockBalance(tx, addr, at)
	if err != nil {
		return false
	}
	err = tx.Commit()
	if err != nil {
		logger.Infof("%v", err)
		return false
	}
	return true
}

func (Bc *Blockchain) Calculationresults(b *block.Block) *block.Block {

	var fnum, tnum uint64
	for _, tx := range b.Txs {
		if bal, ok := b.Txres[tx.From]; ok {
			bal = bal - tx.Amount
			if tx.Script != "" {
				bal = bal - tx.Fee
			}
			b.Txres[tx.From] = bal
			//	logger.Infof("%v==Calculationresults====,%v", b.Height, b.Txres[tx.From])
		} else {
			fnum, _ = Bc.GetBalance(tx.From.AddressToByte())
			bal = fnum - tx.Amount
			b.Txres[tx.From] = bal
			//	logger.Infof("%v==else==Calculationresults====,%v", b.Height, b.Txres[tx.From])
		}
		if bal, ok := b.Txres[tx.To]; ok {
			bal = bal + tx.Amount
			b.Txres[tx.To] = bal
			//	logger.Infof("%v==Calculationresults==b.Txres[tx.To]==,%v", b.Height, b.Txres[tx.To])
		} else {
			tnum, _ = Bc.GetBalance(tx.To.AddressToByte())
			bal = tnum + tx.Amount
			b.Txres[tx.To] = bal
			// logger.Infof("%v==else==Calculationresults==b.Txres[tx.To]==,%v", b.Height, b.Txres[tx.To])
		}

	}
	return b
}

func (Bc *Blockchain) Checkresults(rbl *block.Blocks, Ds, Cm, qtj []byte) bool {

	brs := blocks_change(rbl)
	var bl *block.Block = brs
	bres := Bc.Calculationresults(bl)

	for _, tx := range brs.Txs {
		if tx.Script != "" {
			sc := parser.Parser([]byte(tx.Script))
			e, _ := exec.New(Bc.cdb, sc, string(tx.From.AddressToByte()))
			ert := e.Root()
			if hex.EncodeToString(tx.Root) != hex.EncodeToString(ert) {
				logger.Infof("check==================1 r = %v, er = %v", hex.EncodeToString(tx.Root), hex.EncodeToString(ert))
				return false
			}
		}
		if _, ok := brs.Txres[tx.From]; !ok {
			logger.Infof("check==================2")
			return false
		}
		if _, ok := brs.Txres[tx.To]; !ok {
			logger.Infof("check==================3")
			return false
		}
		if bres.Txres[tx.From] != brs.Txres[tx.From] || bres.Txres[tx.To] != brs.Txres[tx.To] {
			logger.Infof("check==================4==height=", bres.Height)
			logger.Infof("%v,%v  ==== %v,%v", bres.Txres[tx.From], brs.Txres[tx.From], bres.Txres[tx.To], brs.Txres[tx.To])
			return false
		}
	}
	mtx := Distr(rbl.Txs, rbl.Miner, Ds, Cm, qtj, rbl.Height)
	var maptx map[string]uint64
	maptx = make(map[string]uint64)
	for _, mt := range mtx {
		if _, ok := maptx[string(mt.RecAddress)]; !ok {
			maptx[string(mt.RecAddress)] = mt.Amount
			continue
		}
		maptx[string(mt.RecAddress)] += mt.Amount
	}
	//======传过来的
	var oldmaptx map[string]uint64
	oldmaptx = make(map[string]uint64)

	for _, mt := range rbl.FirstTx {
		if _, ok := oldmaptx[string(mt.RecAddress)]; !ok {
			oldmaptx[string(mt.RecAddress)] = mt.Amount
			continue
		}
		oldmaptx[string(mt.RecAddress)] += mt.Amount
	}

	for addr, _ := range maptx {
		if maptx[addr] != oldmaptx[addr] {
			logger.Infof("============maptx[string(addr)] %v != oldmaptx[addr] %v ====", maptx[string(addr)], oldmaptx[addr])
			return false
		}
	}
	for k, _ := range bres.Txres {
		delete(brs.Txres, k)
	}
	for k, _ := range brs.Txres {
		delete(bres.Txres, k)
	}
	for k, _ := range maptx {
		delete(maptx, k)
	}
	return true
}

func blocks_change(b *block.Blocks) *block.Block {

	var bm *block.Block = &block.Block{
		Height:        b.Height,
		PrevBlockHash: b.PrevBlockHash,
		Txs:           b.Txs,
		Root:          b.Root,
		Version:       b.Version,
		Timestamp:     b.Timestamp,
		Hash:          b.Hash,
		Miner:         b.Miner,
		Txres:         make(map[types.Address]uint64),
		FirstTx:       b.FirstTx,
	}

	for _, br := range b.Res {
		bm.Txres[br.Address] = br.Balance

	}

	return bm
}

func (Bc *Blockchain) GetBlockSection(currentHeight, lastHeight uint64) ([]*block.Blocks, error) {
	var bs []*block.Blocks

	for i := lastHeight; i <= currentHeight; i++ {
		hash, err := Bc.db.Mget(miscellaneous.E64func(i), []byte("hash"))
		if err != nil {
			logger.Infof("1111,%v", err)
			return nil, err
		}
		B, err := Bc.db.Get(hash)
		if err != nil {
			logger.Infof("222,%v", err)
			return nil, err
		}

		blcok := &block.Blocks{}
		err = json.Unmarshal(B, blcok)
		if err != nil {
			logger.Infof("333,%v", err)
			return nil, err
		}
		bs = append(bs, blcok)

	}

	return bs, nil
}
