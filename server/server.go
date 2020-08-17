package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"kto/block"
	"kto/blockchain"
	"kto/p2p/node"
	"kto/transaction"
	"kto/txpool"
	"kto/until/logger"
	"strconv"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

type resp struct {
	Address string `json:"address"`
	Hash    string `json:"hash"`
	Num     int    `json:"num"`
	Page    int    `json:"page"`
}
type reqLock struct {
	Address string `json:"address"`
	Pas     string `json:"pas"`
	Amount  string `json:"amount"`
}
type Transaction struct {
	Nonce       uint64 `json:"nonce"`
	BlockNumber uint64 `json:"blocknumber"`
	Amount      uint64 `json:"amount"`
	From        string `json:"from"`
	To          string `json:"to"`
	Hash        string `json:"hash"`
	Signature   string `json:"signature"`
	Time        int64  `json:"time"`
	Script      string `json:"script"`
	Ord         Order  `json:"ord"`
}

type Balance struct {
	Bal uint64 `json:"balance"`
}
type Nonce struct {
	Nonce uint64 `json:"nonce"`
}
type Height struct {
	Height uint64 `json:"height"`
}
type ErrMsg struct {
	Errno  uint   `json:"errno"`
	Errmsg string `json:"errmsg"`
}
type Block struct {
	Height        uint64          `json:"height"`
	PrevBlockHash string          `json:"prevblockhash"`
	Miner         string          `json:"miner"`
	Txs           []Transaction   `json:"txs"`
	Root          string          `json:"root"`
	Version       uint64          `json:"version"`
	Timestamp     int64           `json:"timestamp"`
	Hash          string          `json:"hash"`
	FirstTx       []block.MinerTx `json:"firsttx"`
}

type Order struct {
	Id         string `json:"id"`
	Address    string `json:"address"`
	Price      uint64 `json:"price"`
	Hash       string `json:"hash"`
	Signature  string `json:"signature"`
	Ciphertext string `json:"ciphertext"`
	Tradename  string `json:"tradename"`
	Region     string `json:"region"`
}

type Servers interface {
	Run()
	chukuai()
}

type Server struct {
	r  *fasthttprouter.Router
	Bc *blockchain.Blockchain
	tp *txpool.Txpool
	n  node.Node
}

func New(Bc *blockchain.Blockchain, tps *txpool.Txpool, n node.Node) *Server {
	r := fasthttprouter.New()
	s := &Server{r, Bc, tps, n}
	return s
}

func (s *Server) Gettxpool() *txpool.Txpool {
	return s.tp
}
func (s *Server) Run() {
	f := Logger()
	defer f.Close()
	fmt.Println("Runserver")
	s.r.POST("/ReceTransaction", s.RecepTx)
	s.r.POST("/GetBalancebyAddr", s.GetBalancebyAddr)
	s.r.POST("/GetTxsbyAddr", s.GetTransactionbyAddr)
	s.r.POST("/GetTxbyhash", s.GetTransactionbyhash)
	//s.r.POST("/GetBlockbyHash", s.GetBlockbyHash)
	s.r.POST("/GetBlockbyNum", s.GetBlockbyNum)
	s.r.POST("/GetBlockbyNum_FY", s.GetBlockbyNum_FY)
	s.r.GET("/GetBlock_FY", s.GetBlock_FY)
	// s.r.POST("/GetTx_FY", s.GetTx_FY)
	s.r.POST("/kto/getnonce", s.getNonceHandler)
	s.r.POST("/GetMaxBlockNum", s.GetMaxBlockNum)
	s.r.GET("/GetUSDkto", s.GetUSDkto)
	s.r.GET("/GetUsdtCny", s.GetUsdtCny)

	err := fasthttp.ListenAndServe("0.0.0.0:12344", s.r.Handler)
	if err != nil {
		fmt.Println("start fasthttp fail:", err.Error())
	}
}

type Block_Fy struct {
	Height uint64 `json:"height"`
	Txs    uint64 `json:"txs"`
}

func (s *Server) GetBlock_FY(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respstr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respstr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	h, err := s.Bc.GetMaxBlockHeight()
	if err != nil {
		errorCode = ErrNoBlock
		fmt.Println(err)
		return
	}
	// var bf []Block_Fy
	// var heightLimit int
	// if h > 30 {
	// 	heightLimit = int(h)
	// } else {
	// 	heightLimit = 30
	// }
	// for i := 0; i < heightLimit; i++ {
	// 	var by Block_Fy
	// 	b, err := s.Bc.GetBlockByHeight(h)
	// 	fmt.Println("height:", h)
	// 	if err != nil {
	// 		errorCode = ErrNoBlock
	// 		fmt.Println("get block err:", err)
	// 		return
	// 	}
	// 	by.Txs = uint64(len(b.Txs))
	// 	fmt.Println("141")
	// 	by.Height = h
	// 	bf = append(bf, by)
	// 	h -= 1

	// }

	var latestBlcok Block
	for h > 0 {
		tmpblock, err := s.Bc.GetBlockByHeight(h)
		if err != nil {
			errorCode = ErrNoBlockHeight
			Error("fail to get blcok by height:", err)
			return
		}
		if tmpblock.Txs == nil || len(tmpblock.Txs) == 0 {
			h--
			continue
		}
		latestBlcok.Height = tmpblock.Height
		latestBlcok.Hash = hex.EncodeToString(tmpblock.Hash)
		latestBlcok.PrevBlockHash = hex.EncodeToString(tmpblock.PrevBlockHash)
		latestBlcok.Root = hex.EncodeToString(tmpblock.Root)
		latestBlcok.Timestamp = tmpblock.Timestamp
		latestBlcok.Version = tmpblock.Version
		latestBlcok.Miner = string(tmpblock.Miner)
		latestBlcok.FirstTx = tmpblock.FirstTx
		for _, tx := range tmpblock.Txs {
			var tmpTx Transaction
			tmpTx.Hash = hex.EncodeToString(tx.Hash)
			tmpTx.From = string(tx.From.AddressToByte())
			tmpTx.Amount = tx.Amount
			tmpTx.Nonce = tx.Nonce
			tmpTx.To = string(tx.To.AddressToByte())
			tmpTx.Signature = hex.EncodeToString(tx.Signature)
			tmpTx.Time = tx.Time
			//tmpTx.BlockNumber = tx.BlockNumber   //blocknumber为0
			tmpTx.BlockNumber = tmpblock.Height

			tmpTx.Ord.Id = string(tx.Ord.Id)
			tmpTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
			tmpTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
			tmpTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
			tmpTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
			tmpTx.Ord.Price = tx.Ord.Price
			latestBlcok.Txs = append(latestBlcok.Txs, tmpTx)
		}
		break
	}

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		BF        Block  `json:"bf'`
	}

	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.BF = latestBlcok
	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) GetMaxBlockNum(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respstr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respstr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	h, err := s.Bc.GetMaxBlockHeight()
	if err != nil {
		errorCode = ErrNoBlockHeight
		fmt.Println(err)
		return
	}

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Height    uint64 `json:"height"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Height = h

	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) getNonceHandler(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respstr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respstr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	var reqData resp
	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		fmt.Println("Unmarshal is  failed")
		return
	}

	nonce, err := s.Bc.GetNonce([]byte(reqData.Address))
	if err != nil {
		errorCode = ErrNoNonce
		fmt.Println("GetNonce is failed")
		return
	}

	errorCode = Success
	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Nonce     uint64 `json:"nonce"`
	}
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Nonce = nonce

	respBody, _ = json.Marshal(respData)
	return
}

//接受交易，放入内存
func (s *Server) RecepTx(ctx *fasthttp.RequestCtx) {
	var respBody []byte
	var errorCode int
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()

	reqBody := ctx.PostBody()
	var reqData transaction.Transaction
	err := json.Unmarshal(reqBody, &reqData)
	if err != nil {
		errorCode = ErrJSON
		Error("Unmarshal is faild:", err)
		return
	}
	if reqData.From == reqData.To {
		errorCode = ErrData
		Error("from adderss equal to adderss")
		return
	}

	if err := s.tp.Add(&reqData, s.Bc); err != nil {
		errorCode = ErrtTx
		Error("fail to add tx:", err)
		return
	}
	s.n.Broadcast(&reqData)

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Hash      string `json:"hash"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Hash = hex.EncodeToString(reqData.Hash)

	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) GetTransactionbyAddr(ctx *fasthttp.RequestCtx) {
	var reqData resp
	var respBody []byte
	var errorCode int
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail parse json:", err)
		return
	}

	err, txs := s.Bc.GetTransactionByAddr([]byte(reqData.Address))
	if err != nil {
		errorCode = ErrNoTransaction
		Error("fail to get tx:", err)
		return
	}
	if txs == nil {
		errorCode = ErrNoTransaction
		err = errors.New("tx list is nil")
		Error("no tx:", err)
		return
	}

	var Tx []Transaction
	for _, tx := range txs {
		var tmpTx Transaction
		tmpTx.Hash = hex.EncodeToString(tx.Hash)
		tmpTx.From = string(tx.From.AddressToByte())
		tmpTx.Amount = tx.Amount
		tmpTx.Nonce = tx.Nonce
		tmpTx.To = string(tx.To.AddressToByte())
		tmpTx.Signature = hex.EncodeToString(tx.Signature)
		tmpTx.Time = tx.Time
		tmpTx.BlockNumber = tx.BlockNumber
		tmpTx.Script = tx.Script
		//===
		tmpTx.Ord.Id = string(tx.Ord.Id)
		tmpTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
		tmpTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
		tmpTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
		tmpTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
		tmpTx.Ord.Price = tx.Ord.Price
		tmpTx.Ord.Region = string(tx.Ord.Region)
		tmpTx.Ord.Tradename = string(tx.Ord.Tradename)
		//====
		Tx = append(Tx, tmpTx)

	}
	var respData struct {
		ErrorCode       int           `json:"errorcode"`
		ErrorMsg        string        `json:"errormsg"`
		TransactionList []Transaction `json:"transactionlist"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.TransactionList = Tx

	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) GetTransactionbyhash(ctx *fasthttp.RequestCtx) {
	var reqData resp
	var respBody []byte
	var errorCode int
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail to parse json:", err)
		return
	}

	hash, err := hex.DecodeString(reqData.Hash)
	if err != nil {
		errorCode = ErrData
		Error("fail to parse hash:", err)
		return
	}

	tx, err := s.Bc.GetTransaction(hash)
	if err != nil {
		errorCode = ErrNoTxByHash
		Error("fail to get tx:", err)
		return
	}

	var retTx Transaction
	retTx.Hash = hex.EncodeToString(tx.Hash)
	retTx.From = string(tx.From.AddressToByte())
	retTx.Amount = tx.Amount
	retTx.Nonce = tx.Nonce
	retTx.To = string(tx.To.AddressToByte())
	retTx.Signature = hex.EncodeToString(tx.Signature)
	retTx.Time = tx.Time
	retTx.BlockNumber = tx.BlockNumber
	retTx.Script = tx.Script
	//==
	retTx.Ord.Id = string(tx.Ord.Id)
	retTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
	retTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
	retTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
	retTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
	retTx.Ord.Price = tx.Ord.Price
	retTx.Ord.Region = string(tx.Ord.Region)
	retTx.Ord.Tradename = string(tx.Ord.Tradename)

	var respData struct {
		ErrorCode   int         `json:"errorcode"`
		ErroeMsg    string      `json:"errormsg"`
		Transaction Transaction `json:"transaction"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErroeMsg = ErrorMap[errorCode]
	respData.Transaction = retTx

	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) GetBalancebyAddr(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()

	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	reqBody := ctx.PostBody()
	var reqData resp
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail to parse json:", err)
		return
	}

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Balance   uint64 `jaon:"balance"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	fmt.Println("[]byte(reqData.Address)=====", []byte(reqData.Address))
	fmt.Println(reqData.Address)
	balance, err := s.Bc.GetBalance([]byte(reqData.Address))
	if err != nil {
		respData.Balance = 0
		respBody, _ = json.Marshal(respData)
		Error("get balance error:", err)
		return
	}

	respData.Balance = balance
	respBody, _ = json.Marshal(respData)
	return
}

func (s *Server) GetBlockbyHash(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()

	var reqData resp
	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail to parse json:", err)
		return
	}
	h, _ := hex.DecodeString(reqData.Hash)
	b, err := s.Bc.GetBlockByHash(h)
	if err != nil {
		errorCode = ErrNoBlock
		Error("fail to get block:", err)
		return
	}
	Debug(b)
	var bl Block
	var Tx []Transaction
	if len(b.Txs) != 0 {
		for _, tx := range b.Txs {
			var tmpTx Transaction
			tmpTx.Hash = hex.EncodeToString(tx.Hash)
			tmpTx.From = string(tx.From.AddressToByte())
			tmpTx.Amount = tx.Amount
			tmpTx.Nonce = tx.Nonce
			tmpTx.To = string(tx.To.AddressToByte())
			tmpTx.Signature = hex.EncodeToString(tx.Signature)
			tmpTx.Time = tx.Time
			tmpTx.BlockNumber = tx.BlockNumber
			Tx = append(Tx, tmpTx)

		}
		bl.Txs = Tx
	} else {
		bl.Txs = nil
	}

	bl.Height = b.Height
	bl.Hash = hex.EncodeToString(b.Hash)
	bl.PrevBlockHash = hex.EncodeToString(b.PrevBlockHash)
	bl.Root = hex.EncodeToString(b.Root)
	bl.Timestamp = b.Timestamp
	bl.Version = b.Version
	bl.Miner = string(b.Miner)
	bl.FirstTx = b.FirstTx

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Block     Block  `json:"block"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Block = bl

	respBody, _ = json.Marshal(respData)
	return

}
func (s *Server) GetBlockbyNum(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	var reqData resp
	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail to parse json:", err)
		return
	}

	fmt.Println("num = ", reqData.Num)
	b, err := s.Bc.GetBlockByHeight(uint64(reqData.Num))
	if err != nil {
		errorCode = ErrNoBlock
		Error("fail to get block:", err)
		return
	}

	Debug(b)
	var bl Block
	var Tx []Transaction
	if len(b.Txs) != 0 {
		for _, tx := range b.Txs {
			var tmpTx Transaction
			tmpTx.Hash = hex.EncodeToString(tx.Hash)
			tmpTx.From = string(tx.From.AddressToByte())
			tmpTx.Amount = tx.Amount
			tmpTx.Nonce = tx.Nonce
			tmpTx.To = string(tx.To.AddressToByte())
			tmpTx.Signature = hex.EncodeToString(tx.Signature)
			tmpTx.Time = tx.Time
			tmpTx.BlockNumber = tx.BlockNumber
			tmpTx.Script = tx.Script

			tmpTx.Ord.Id = string(tx.Ord.Id)
			tmpTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
			tmpTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
			tmpTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
			tmpTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
			tmpTx.Ord.Price = tx.Ord.Price
			tmpTx.Ord.Region = string(tx.Ord.Region)
			tmpTx.Ord.Tradename = string(tx.Ord.Tradename)

			Tx = append(Tx, tmpTx)
		}
		bl.Txs = Tx
	} else {
		bl.Txs = nil
	}

	bl.Height = b.Height
	bl.Hash = hex.EncodeToString(b.Hash)
	bl.PrevBlockHash = hex.EncodeToString(b.PrevBlockHash)
	bl.Root = hex.EncodeToString(b.Root)
	bl.Timestamp = b.Timestamp
	bl.Version = b.Version
	bl.Miner = string(b.Miner)
	bl.FirstTx = b.FirstTx
	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Block     Block  `json:"block"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Block = bl

	respBody, _ = json.Marshal(respData)
	return
}

type block_fy struct {
	TxList []Transaction `json:"txlsit"`
	Total  uint64        `json:"total"`
	Height uint64        `json:"height"`
}

func (s *Server) GetBlockbyNum_FY(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			respBody = []byte(respStr)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	var reqData resp
	reqBody := ctx.PostBody()
	if err := json.Unmarshal(reqBody, &reqData); err != nil {
		errorCode = ErrJSON
		Error("fail to parse json:", err)
		return
	}
	if reqData.Num < 0 || reqData.Page < 0 {
		errorCode = ErrData
		Errorf("data error:number:%d,page:%d", reqData.Num, reqData.Page)
		return
	}
	maxHeight, err := s.Bc.GetMaxBlockHeight()
	if err != nil {
		errorCode = ErrNoBlockHeight
		Error("fail to get block height:", err)
		return
	}

	blocknumber := int(maxHeight) - (reqData.Page-1)*reqData.Num
	if blocknumber <= 0 {
		errorCode = ErrData
		fmt.Println("Fy number:", blocknumber)
		Error("error data", blocknumber)
		return
	}

	var blockList []Block
	Errorf("blocknumber:%d,num:%d,page:%d\n", blocknumber, reqData.Num, reqData.Page)
	for i := blocknumber; i > 0 && i > (blocknumber-reqData.Num); i-- {
		fmt.Println("FY:%d---------------------------", i)
		block, err := s.Bc.GetBlockByHeight(uint64(i))
		if err != nil {
			//errorCode = ErrNoBlock
			Error("fail to get blcok:", err)
			break
		}
		Errorf("block:", block)
		var tmpBlock Block
		//var Tx []Transaction
		for _, tx := range block.Txs {
			var tmpTx Transaction
			tmpTx.Hash = hex.EncodeToString(tx.Hash)
			tmpTx.From = string(tx.From.AddressToByte())
			tmpTx.Amount = tx.Amount
			tmpTx.Nonce = tx.Nonce
			tmpTx.To = string(tx.To.AddressToByte())
			tmpTx.Signature = hex.EncodeToString(tx.Signature)
			tmpTx.Time = tx.Time
			tmpTx.BlockNumber = tx.BlockNumber
			tmpTx.Script = tx.Script

			tmpTx.Ord.Id = string(tx.Ord.Id)
			tmpTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
			tmpTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
			tmpTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
			tmpTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
			tmpTx.Ord.Price = tx.Ord.Price
			tmpBlock.Txs = append(tmpBlock.Txs, tmpTx)
		}
		tmpBlock.Height = block.Height
		tmpBlock.Hash = hex.EncodeToString(block.Hash)
		tmpBlock.PrevBlockHash = hex.EncodeToString(block.PrevBlockHash)
		tmpBlock.Root = hex.EncodeToString(block.Root)
		tmpBlock.Timestamp = block.Timestamp
		tmpBlock.Version = block.Version
		tmpBlock.Miner = string(block.Miner)
		tmpBlock.FirstTx = block.FirstTx
		blockList = append(blockList, tmpBlock)
	}

	var respData struct {
		ErrorCode int    `json:"errorcode"`
		ErrorMsg  string `json:"errormsg"`
		Data      struct {
			Total     uint64  `json:"total"`
			BlockList []Block `json:"blocklist"`
		} `json:"block"`
	}
	errorCode = Success
	respData.ErrorCode = errorCode
	respData.ErrorMsg = ErrorMap[errorCode]
	respData.Data.BlockList = blockList
	respData.Data.Total = maxHeight
	// fmt.Println("NUm = ", reqData.Num)
	// fmt.Println("PAGE = ", reqData.Page)
	// b, err := s.Bc.GetBlockByHeight(uint64(reqData.Num))
	// if err != nil {
	// 	errorCode = ErrNoBlock
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(b)

	// if len(b.Txs) != 0 {
	// 	for _, tx := range b.Txs {
	// 		var tmpTx Transaction
	// 		tmpTx.Hash = hex.EncodeToString(tx.Hash)
	// 		tmpTx.From = string(tx.From.AddressToByte())
	// 		tmpTx.Amount = tx.Amount
	// 		tmpTx.Nonce = tx.Nonce
	// 		tmpTx.To = string(tx.To.AddressToByte())
	// 		tmpTx.Signature = hex.EncodeToString(tx.Signature)
	// 		tmpTx.Time = tx.Time
	// 		tmpTx.BlockNumber = tx.BlockNumber

	// 		tmpTx.Ord.Id = string(tx.Ord.Id)
	// 		tmpTx.Ord.Hash = hex.EncodeToString(tx.Ord.Hash)
	// 		tmpTx.Ord.Signature = hex.EncodeToString(tx.Ord.Signature)
	// 		tmpTx.Ord.Ciphertext = hex.EncodeToString(tx.Ord.Ciphertext)
	// 		tmpTx.Ord.Address = string(tx.Ord.Address.AddressToByte())
	// 		tmpTx.Ord.Price = tx.Ord.Price
	// 		Tx = append(Tx, tmpTx)

	// 	}

	// 	switch reqData.Page {
	// 	case 1:
	// 		if len(Tx) > 30 {
	// 			Tx = Tx[:30]
	// 		} else {
	// 			Tx = Tx[:len(Tx)]
	// 		}

	// 	case 2:
	// 		if len(Tx) > 60 {
	// 			Tx = Tx[30:60]
	// 		} else {
	// 			Tx = Tx[30:len(Tx)]
	// 		}
	// 	case 3:
	// 		if len(Tx) > 90 {
	// 			Tx = Tx[60:90]
	// 		} else {
	// 			Tx = Tx[60:len(Tx)]
	// 		}
	// 	case 4:
	// 		if len(Tx) > 120 {
	// 			Tx = Tx[90:120]
	// 		} else {
	// 			Tx = Tx[90:len(Tx)]
	// 		}
	// 	case 5:
	// 		if len(Tx) > 149 {
	// 			Tx = Tx[120:149]
	// 		} else {
	// 			Tx = Tx[120:len(Tx)]
	// 		}

	// 	default:
	// 		break
	// 	}

	// }

	// var respData struct {
	// 	ErrorCode int      `json:"errorcode"`
	// 	ErrorMsg  string   `json:"errormsg"`
	// 	BT        block_fy `json:"bt"`
	// }
	// var bt block_fy
	// bt.TxList = Tx
	// bt.Total = uint64(len(bt.TxList))
	// bt.Height = b.Height

	// errorCode = Success
	// respData.ErrorCode = errorCode
	// respData.ErrorMsg = ErrorMap[errorCode]
	// respData.BT = bt

	respBody, _ = json.Marshal(respData)
	return
}

// func (s *Server) Chukuai() {
// 	time.Sleep(time.Second * 5)
// 	s.tp.Pendings(s.Bc)
// 	b, _ := s.Bc.NewBlock(s.tp.Pending)
// 	s.Bc.AddBlock(b, []byte("2o76U4LXhTwjir8gqpShsPxHio7iFb3MR2WEEgef2y9v"))
// 	data, err := json.Marshal(b)
// 	if err != nil {
// 		log.Fatalf("JSON marshaling failed: %s", err)
// 	}
// 	fmt.Printf("%s\n", data)
// 	s.tp.Pending = nil
// }

func (s *Server) GetUSDkto(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			// respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			// respBody = []byte(respStr)
			respBody = []byte(`{"success":true}`)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	url := "https://www.bcone.vip/api/market/tickers/ticker?symbol=usdt_kto"
	_, respBody, err := fasthttp.Get(nil, url)
	if err != nil {
		errorCode = ErrData
		return
	}
	errorCode = Success
}

func (s *Server) GetUsdtCny(ctx *fasthttp.RequestCtx) {
	var errorCode int
	var respBody []byte
	defer func() {
		if errorCode != Success {
			// respStr := fmt.Sprintf(`{"errorcode":%d,"errormsg":"%s"}`, errorCode, ErrorMap[errorCode])
			// respBody = []byte(respStr)
			respBody = []byte(`{"flag":0,"isSuccess":false,"data":{},"message":"数据错误"}`)
		}
		ctx.Write(respBody)
	}()
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Request.Header.Add("Access-Control-Allow-Headers", "Content-Type")
	ctx.Request.Header.Set("content-type", "application/json")

	req := &fasthttp.Request{}
	req.AppendBody([]byte("{}"))
	req.SetRequestURI("https://www.bcone.vip/api/currency/currency/getUsdtCny")
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")

	resp := &fasthttp.Response{}

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		errorCode = ErrData
		return
	}
	errorCode = Success
	respBody = resp.Body()
	return
}

func (s *Server) PackBlock(minaddr, Ds, Cm, QTJ []byte) (*block.Block, error) {
	logger.Infof("Package blocks from txpool...")
	s.tp.Pendings(s.Bc)
	b, err := s.Bc.NewBlock(s.tp.Pending, minaddr, Ds, Cm, QTJ)
	if err != nil {
		return nil, err
	}
	s.RemovePending()
	return b, nil
}

// func (s *Server) CommitBlock(b *block.Blocks, Ds, Cm []byte) error {
func (s *Server) CommitBlock(b *block.Blocks, minaddr []byte) error {
	logger.Infof("server: Start Commit Block.\n")
	//	b.Miner = []byte(mineraddr)
	// err := s.Bc.AddBlock(b, Ds, Cm)
	err := s.Bc.AddBlock(b, minaddr)
	if err != nil {
		return err
	}

	logger.Infof("server: End Commit Block.\n")
	return nil
}

func (s *Server) RemovePending() {

	s.tp.Pending = nil

}

// func (s *Server) Chukuai()  {
// 	b := s.PackBlock()
// 	s.CommitBlock(b)
// }

const PasLock = "myxwxhqutjqc"

func (s *Server) LockAmount(ctx *fasthttp.RequestCtx) {

	a := ctx.PostBody()
	var v reqLock
	json.Unmarshal(a, &v)
	if v.Pas != PasLock {
		fmt.Fprintf(ctx, "%s", "lock failed")
	}
	num, _ := strconv.ParseInt(v.Amount, 10, 64)
	s.Bc.LockBalance(v.Address, uint64(num))

}
