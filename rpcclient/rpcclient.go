package rpcclient

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"kto/blockchain"
	"kto/p2p/node"
	"kto/rpcclient/message"
	"kto/transaction"
	"kto/txpool"
	"kto/types"
	"kto/until"
	"kto/until/miscellaneous"
	"net"
	"os"
	"strconv"

	"golang.org/x/crypto/sha3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Rpc struct {
	Bc *blockchain.Blockchain
	tp *txpool.Txpool
	message.UnimplementedGreeterServer
	n node.Node
}

func Run_rpc(Bc *blockchain.Blockchain, tp *txpool.Txpool, n node.Node) {
	logFile := Logger()
	defer logFile.Close()

	Debug("runrpc")
	s := &Rpc{Bc: Bc, tp: tp, n: n}
	lis, err := net.Listen("tcp", "0.0.0.0:8544")
	if err != nil {
		Error("failed to listen:", err)
		os.Exit(-1)
	}
	//server := grpc.NewServer()
	server := grpc.NewServer(grpc.UnaryInterceptor(ipInterceptor))
	message.RegisterGreeterServer(server, s)
	server.Serve(lis)
}

func (s *Rpc) GetBalance(ctx context.Context, in *message.ReqBalance) (*message.ResBalance, error) {

	balance, _ := s.Bc.GetBalance([]byte(in.Address))
	return &message.ResBalance{Balnce: balance}, nil
}

func (s *Rpc) GetBlockByNum(ctx context.Context, v *message.ReqBlockByNumber) (*message.RespBlock, error) {

	b, err := s.Bc.GetBlockByHeight(v.Height)
	if err != nil {
		Error("fail to get block:", err)
		return nil, err
	}

	var respdata *message.RespBlock = new(message.RespBlock)
	respdata.Txs = make([]*message.Tx, len(b.Txs))
	for i := 0; i < len(respdata.Txs); i++ {
		respdata.Txs[i] = &message.Tx{Order: &message.Order{}}
	}

	for i, tx := range b.Txs {
		if tx == nil {
			continue
		}
		if tx.Hash != nil {
			respdata.Txs[i].Hash = hex.EncodeToString(tx.Hash)
		}
		if tx.Signature != nil {
			respdata.Txs[i].Signature = hex.EncodeToString(tx.Signature)
		}
		respdata.Txs[i].From = string(tx.From.AddressToByte())
		respdata.Txs[i].Amount = tx.Amount
		respdata.Txs[i].Nonce = tx.Nonce
		respdata.Txs[i].To = string(tx.To.AddressToByte())
		respdata.Txs[i].Time = tx.Time
		respdata.Txs[i].Script = tx.Script
	}

	respdata.Height = b.Height
	respdata.Hash = hex.EncodeToString(b.Hash)
	respdata.PrevBlockHash = hex.EncodeToString(b.PrevBlockHash)
	respdata.Root = hex.EncodeToString(b.Root)
	respdata.Timestamp = b.Timestamp
	respdata.Version = b.Version
	respdata.Miner = string(b.Miner)
	return respdata, nil
}

func (s *Rpc) GetBlockByHash(ctx context.Context, v *message.ReqBlockByHash) (*message.RespBlock, error) {
	h, _ := hex.DecodeString(v.Hash)
	b, err := s.Bc.GetBlockByHash(h)
	if err != nil {
		Error("fail to get blcok:", err)
		return nil, err
	}

	var respdata *message.RespBlock = new(message.RespBlock)
	respdata.Txs = make([]*message.Tx, len(b.Txs))
	for i := 0; i < len(respdata.Txs); i++ {
		respdata.Txs[i] = &message.Tx{Order: &message.Order{}}
	}

	for i, tx := range b.Txs {
		if tx == nil {
			continue
		}
		if tx.Hash != nil {
			respdata.Txs[i].Hash = hex.EncodeToString(tx.Hash)
		}
		if tx.Signature != nil {
			respdata.Txs[i].Signature = hex.EncodeToString(tx.Signature)
		}
		respdata.Txs[i].From = string(tx.From.AddressToByte())
		respdata.Txs[i].Amount = tx.Amount
		respdata.Txs[i].Nonce = tx.Nonce
		respdata.Txs[i].To = string(tx.To.AddressToByte())
		respdata.Txs[i].Time = tx.Time
		respdata.Txs[i].Script = tx.Script
	}

	respdata.Height = b.Height
	respdata.Hash = hex.EncodeToString(b.Hash)
	respdata.PrevBlockHash = hex.EncodeToString(b.PrevBlockHash)
	respdata.Root = hex.EncodeToString(b.Root)
	respdata.Timestamp = b.Timestamp
	respdata.Version = b.Version
	respdata.Miner = string(b.Miner)
	return respdata, nil
}

func (s *Rpc) GetTxsByAddr(ctx context.Context, in *message.ReqTx) (*message.ResposeTxs, error) {
	err, tx_s := s.Bc.GetTransactionByAddr([]byte(in.Address))
	if err != nil {
		Error("fail to get tx:", err)
		return nil, err
	}
	if tx_s == nil {
		Error("tx lsit is nil")
		return nil, err
	}
	var respData *message.ResposeTxs = &message.ResposeTxs{}
	//var Txs message.ResTx
	for _, txs := range tx_s {
		var tmpTx *message.Tx = &message.Tx{}
		tmpTx.Hash = hex.EncodeToString(txs.Hash)
		tmpTx.From = string(txs.From.AddressToByte())
		tmpTx.Amount = txs.Amount
		tmpTx.Nonce = txs.Nonce
		tmpTx.To = string(txs.To.AddressToByte())
		tmpTx.Signature = hex.EncodeToString(txs.Signature)
		tmpTx.Time = txs.Time
		tmpTx.Script = txs.Script

		if len(txs.Ord.Id) != 0 {
			tmpTx.Order = &message.Order{}
			tmpTx.Order.Id = string(txs.Ord.Id)
			tmpTx.Order.Address = string(txs.Ord.Address.AddressToByte())
			tmpTx.Order.Price = txs.Ord.Price
			tmpTx.Order.Hash = until.Encode(txs.Ord.Hash)
			tmpTx.Order.Signature = until.Encode(txs.Ord.Signature)
			tmpTx.Order.Ciphertext = string(txs.Ord.Ciphertext)
			tmpTx.Order.Region = string(txs.Ord.Region)
			tmpTx.Order.Tradename = string(txs.Ord.Tradename)
		}
		respData.Txs = append(respData.Txs, tmpTx)
	}
	//data, _ := json.MarshalIndent(Txs.Txs, "", " ")
	//var resTx message.ResposeTxs = message.ResposeTxs{Txs: data}
	//Debugf("Greeting: %s", data)

	return respData, nil
}

func (s *Rpc) GetTxByHash(ctx context.Context, in *message.ReqTxByHash) (*message.Tx, error) {

	hash, err := hex.DecodeString(in.Hash)
	if err != nil {
		Error("fail to parse hash:", err)
		return nil, err
	}
	tx, err := s.Bc.GetTransaction(hash)
	if err != nil {
		Error("fail to get tx", err)
		return nil, err
	}
	respTx := &message.Tx{}
	respTx.Hash = hex.EncodeToString(tx.Hash)
	respTx.From = string(tx.From.AddressToByte())
	respTx.Amount = tx.Amount
	respTx.Nonce = tx.Nonce
	respTx.To = string(tx.To.AddressToByte())
	respTx.Signature = hex.EncodeToString(tx.Signature)
	respTx.Time = tx.Time
	respTx.Script = tx.Script

	if len(tx.Ord.Id) != 0 {
		respTx.Order = &message.Order{}
		respTx.Order.Id = string(tx.Ord.Id)
		respTx.Order.Address = string(tx.Ord.Address.AddressToByte())
		respTx.Order.Price = tx.Ord.Price
		respTx.Order.Hash = until.Encode(tx.Ord.Hash)
		respTx.Order.Signature = until.Encode(tx.Ord.Signature)
		respTx.Order.Ciphertext = string(tx.Ord.Ciphertext)
		respTx.Order.Region = string(tx.Ord.Region)
		respTx.Order.Tradename = string(tx.Ord.Tradename)
	}
	return respTx, nil
}

func (s *Rpc) GetAddressNonceAt(ctx context.Context, in *message.ReqNonce) (*message.ResposeNonce, error) {

	nonce, err := s.Bc.GetNonce([]byte(in.Address))
	if err != nil {
		Error("fail to get nonce:", err)
		return nil, err
	}
	var N message.ResposeNonce
	N.Nonce = nonce
	return &N, nil
}

func (s *Rpc) SendTransaction(ctx context.Context, in *message.ReqTransaction) (*message.ResTransaction, error) {

	if in.From == in.To {
		Error("request data error")
		return nil, grpc.Errorf(codes.InvalidArgument, "data error")
	}

	from := types.BytesToAddress([]byte(in.From))
	to := types.BytesToAddress([]byte(in.To))

	priv := until.Decode(in.Priv)
	if len(priv) != 64 {
		Errorf("priv length error：%d", len(priv))
		return nil, grpc.Errorf(codes.InvalidArgument, "data error")
	}

	tx := transaction.New()
	tx = tx.NewTransaction(in.Nonce, in.Amount, from, to, "")
	if in.Order != nil {
		//var orderAddress types.Address
		// orderId := []byte(in.Order.Id)
		// orderPrice := in.Order.Price
		// orderCiphertext, _ := hex.DecodeString(in.Order.Ciphertext)

		if len(in.Order.Address) == types.Lenthaddr {
			for i, v := range []byte(in.Order.Address) {
				tx.Ord.Address[i] = v
			}
			var err error
			tx.Ord.Id = []byte(in.Order.Id)
			//tx.Ord.Address = orderAddress
			tx.Ord.Price = in.Order.Price
			tx.Ord.Ciphertext, err = hex.DecodeString(in.Order.Ciphertext)
			if err != nil {
				Error("ciphertext error:", err)
				return nil, grpc.Errorf(codes.InvalidArgument, "data error")
			}
			tx.Ord.Hash, err = hex.DecodeString(in.Order.Hash)
			if err != nil {
				Error("hash error:", err)
				return nil, grpc.Errorf(codes.InvalidArgument, "data error")
			}
			tx.Ord.Signature, err = hex.DecodeString(in.Order.Signature)
			if err != nil {
				Error("Signature error:", err)
				return nil, grpc.Errorf(codes.InvalidArgument, "data error")
			}
			tx.Ord.Region = in.Order.Region
			tx.Ord.Tradename = in.Order.Tradename
		}
	}
	tx = tx.SignTx(priv)

	err := s.tp.Add(tx, s.Bc)
	if err != nil {
		Error("Add tx for txpool error:", err)
		return nil, err
	}

	s.n.Broadcast(tx)
	hash := hex.EncodeToString(tx.Hash)

	return &message.ResTransaction{Hash: hash}, nil
}
func (s *Rpc) SendSignedTransaction(ctx context.Context, in *message.ReqSignedTransaction) (*message.RespSignedTransaction, error) {

	if in.From == in.To {
		Error("request data error")
		return nil, grpc.Errorf(codes.InvalidArgument, "data error")
	}

	from := types.BytesToAddress([]byte(in.From))
	to := types.BytesToAddress([]byte(in.To))

	tx := &transaction.Transaction{
		From:      from,
		To:        to,
		Nonce:     in.Nonce,
		Amount:    in.Amount,
		Time:      in.Time,
		Hash:      in.Hash,
		Signature: in.Signature,
	}

	err := s.tp.Add(tx, s.Bc)
	if err != nil {
		Error("Add tx for txpool error:", err)
		return nil, grpc.Errorf(codes.InvalidArgument, "parameter error")
	}

	s.n.Broadcast(tx)
	hash := hex.EncodeToString(tx.Hash)

	return &message.RespSignedTransaction{Hash: hash}, nil
}

func (s *Rpc) CreateContract(ctx context.Context, in *message.ReqTokenCreate) (*message.RespTokenCreate, error) {

	if in.From == in.To {
		err := errors.New("request data error")
		Error(err)
		return nil, err
	}

	from := types.BytesToAddress([]byte(in.From))
	to := types.BytesToAddress([]byte(in.To))

	priv := until.Decode(in.Priv)
	if len(priv) != 64 {
		err := errors.New("priv error")
		Errorf("priv length error：%d", len(priv))
		return nil, err
	}

	tx := transaction.New()

	//"new \"abc\" 1000000000"
	At := strconv.FormatUint(in.Total, 10)
	//script := "new" + '\"' + in.Symbol +'\"' + in.Total

	script := fmt.Sprintf("new \"%s\" %s", in.Symbol, At)
	tx = tx.Newtoken(in.Nonce, uint64(500001), in.Fee, from, to, script)

	tx = tx.SignTx(priv)

	err := s.tp.Add(tx, s.Bc)
	if err != nil {
		Error(err)
		return nil, err
	}

	s.n.Broadcast(tx)
	hash := hex.EncodeToString(tx.Hash)

	return &message.RespTokenCreate{Hash: hash}, nil
}

func (s *Rpc) MintToken(ctx context.Context, in *message.ReqTokenCreate) (*message.RespTokenCreate, error) {

	if in.From == in.To {
		err := errors.New("request data error")
		Error(err)
		return nil, err
	}

	from := types.BytesToAddress([]byte(in.From))
	to := types.BytesToAddress([]byte(in.To))

	priv := until.Decode(in.Priv)
	if len(priv) != 64 {
		err := errors.New("priv error")
		Errorf("priv length error：%d", len(priv))
		return nil, err
	}

	tx := transaction.New()

	//"new \"abc\" 1000000000"
	At := strconv.FormatUint(in.Total, 10)
	//script := "new" + '\"' + in.Symbol +'\"' + in.Total

	script := fmt.Sprintf("mint \"%s\" %s", in.Symbol, At)
	tx = tx.Newtoken(in.Nonce, uint64(500001), in.Fee, from, to, script)

	tx = tx.SignTx(priv)

	err := s.tp.Add(tx, s.Bc)
	if err != nil {
		Error(err)
		return nil, err
	}

	s.n.Broadcast(tx)
	hash := hex.EncodeToString(tx.Hash)

	return &message.RespTokenCreate{Hash: hash}, nil
}

func (s *Rpc) SendToken(ctx context.Context, in *message.ReqTokenTransaction) (*message.RespTokenTransaction, error) {

	if in.From == in.To {
		err := errors.New("request data error")
		Error(err)
		return nil, err
	}

	from := types.BytesToAddress([]byte(in.From))
	to := types.BytesToAddress([]byte(in.To))

	priv := until.Decode(in.Priv)
	if len(priv) != 64 {
		err := errors.New("priv error")
		Errorf("priv length error：%d", len(priv))
		return nil, err
	}

	tx := transaction.New()
	At := strconv.FormatUint(in.TokenAmount, 10)
	//"transfer \"abc\" 10 \"to\""
	script := fmt.Sprintf("transfer \"%s\" %s \"%s\"", in.Symbol, At, in.To)
	//tx = tx.NewTransaction(in.Nonce, in.Amount, from, to, script)
	tx = tx.Newtoken(in.Nonce, in.Amount, in.Fee, from, to, script)
	tx = tx.SignTx(priv)

	err := s.tp.Add(tx, s.Bc)
	if err != nil {
		Error(err)
		return nil, err
	}

	s.n.Broadcast(tx)
	hash := hex.EncodeToString(tx.Hash)

	return &message.RespTokenTransaction{Hash: hash}, nil
}

func (s *Rpc) GetBalanceToken(ctx context.Context, in *message.ReqTokenBalance) (*message.RespTokenBalance, error) {

	balance, err := s.Bc.GetTokenBalance([]byte(in.Address), []byte(in.Symbol))
	if err != nil {
		Error("fail to get balance:", err)
		return nil, err
	}

	return &message.RespTokenBalance{Balnce: balance}, nil
}

func (s *Rpc) SetLockBalance(ctx context.Context, in *message.ReqLockBalance) (*message.RespLockBalance, error) {
	if ok := s.Bc.LockBalance(in.Address, in.Amount); !ok {
		return &message.RespLockBalance{Status: false}, nil
	}
	return &message.RespLockBalance{Status: true}, nil
}

func (s *Rpc) SetUnlockBalance(ctx context.Context, in *message.ReqUnlockBalance) (*message.RespUnlockBalance, error) {
	if ok := s.Bc.Unlockbalance(in.Address, in.Amount); !ok {
		return &message.RespUnlockBalance{Status: false}, nil
	}
	return &message.RespUnlockBalance{Status: true}, nil
}

func (s *Rpc) CreateAddr(ctx context.Context, in *message.ReqCreateAddr) (*message.RespCreateAddr, error) {
	var addr, prickey string
	for {
		pubKey, privKey, err := until.Generprivkey()
		if err != nil {
			Error("fail to gener priv key:", err)
			return nil, err
		}
		addr = until.PubtoAddr(pubKey)
		prickey = until.Encode(privKey)
		if len(addr) == 47 {
			break
		}

	}

	return &message.RespCreateAddr{Address: addr, Privkey: prickey}, nil
}

//GetMaxBlockNumber 获取最大的块号
func (s *Rpc) GetMaxBlockNumber(ctx context.Context, in *message.ReqMaxBlockNumber) (*message.RespMaxBlockNumber, error) {

	maxHeight, err := s.Bc.GetMaxBlockHeight()
	if err != nil {
		Error("fail to get max height:", err)
		return nil, err
	}
	return &message.RespMaxBlockNumber{MaxNumber: maxHeight}, nil
}

//GetAddrByPriv 通过私钥获取地址
func (s *Rpc) GetAddrByPriv(ctx context.Context, in *message.ReqAddrByPriv) (*message.RespAddrByPriv, error) {
	privBytes := until.Decode(in.Priv)
	if len(privBytes) != 64 {
		err := errors.New("private key error")
		Error(err)
		return nil, err
	}
	addr := until.PubtoAddr(privBytes[32:])
	return &message.RespAddrByPriv{Addr: addr}, nil
}

//GetFrozenAssets 获取冻结资产
func (s *Rpc) GetFrozenAssets(ctx context.Context, in *message.ReqFrozenAssets) (*message.RespFrozenAssets, error) {

	frozenAssets, _ := s.Bc.GetLockBalance(in.Addr)
	return &message.RespFrozenAssets{FrozenAssets: frozenAssets}, nil
}

//SignOrd 签名
func (s *Rpc) SignOrd(ctx context.Context, in *message.ReqSignOrd) (*message.RespSignOrd, error) {
	if in.Order == nil || len(in.Order.Id) == 0 || len(in.Order.Address) == 0 || len(in.Order.Ciphertext) == 0 ||
		in.Order.Price == 0 || len(in.Priv) == 0 {
		return nil, errors.New("data error")
	}

	id, err := hex.DecodeString(in.Order.Id)
	if err != nil {
		return nil, err
	}
	// var orderAddress types.Address
	// for i, v := range []byte(in.Order.Address) {
	// 	orderAddress[i] = v
	// }
	price := miscellaneous.E64func(in.Order.Price)
	ciphertext, err := hex.DecodeString(in.Order.Ciphertext)
	if err != nil {
		return nil, err
	}

	var hashBytes []byte
	hashBytes = append(hashBytes, price...)
	hashBytes = append(hashBytes, id...)
	hashBytes = append(hashBytes, ciphertext...)
	hashBytes = append(hashBytes, []byte(in.Order.Address)...)
	hash32 := sha3.Sum256(hashBytes)
	Hash := hex.EncodeToString(hash32[:])
	pri := ed25519.PrivateKey(until.Decode(in.Priv))
	signatures := ed25519.Sign(pri, hash32[:])
	Signature := hex.EncodeToString(signatures)
	return &message.RespSignOrd{Hash: Hash, Signature: Signature}, nil
}
