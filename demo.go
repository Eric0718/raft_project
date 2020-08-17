// import (
// 	"fmt"
// 	"kto/transaction"
// 	"kto/types"
// 	"kto/until"
// )

// func main() {
// 	l := len("Kto72tzGAwYH7dHGbEH4yiz5gxWSqq9fRDSxXwsJPX98y25")
// 	fmt.Println(l)
// 	from_byte := []byte("Kto9sWkzypDGvxfgcXu5eXJrRzwtX9rG1ftPwQ2NMw3TraX")

// 	to_byte := []byte("Kto72tzGAwYH7dHGbEH4yiz5gxWSqq9fRDSxXwsJPX98y25")

// 	from_pri := until.Decode("2TtUBFZNELwKB4oeo4heRs3mZezaU84ujaHdVgCrki1jHhqn3yQYTaP92BrgrxeadEsStVE2f4aPL9ETsCeRGZjq")

// 	from := types.BytesToAddress(from_byte)
// 	to := types.BytesToAddress(to_byte)
// 	tx := transaction.New()

// 	tx = tx.NewTransaction(uint64(2), 600000, from, to, "")
// 	tx = tx.SignTx(from_pri)
// 	// tx = tx.NewOrder([]byte("123456"), []byte("qwertyuioiop"), from_byte, uint64(100))
// 	// tx = tx.SignOrder(from_pri)
// 	hash, err := tx.SendTransaction()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Println(hash)
// 	// slice = append(slice, hash)

// }

// addr= 72tzGAwYH7dHGbEH4yiz5gxWSqq9fRDSxXwsJPX98y25
// private= VuNP9drXQAWKSSNZnXAcCwz9adZsDvWA6oXUfvzfJvWtZ1Uen5mSiHoYUoxBQZAybiPeWi264VcMpCAaukTxeS1

// addr= 9sWkzypDGvxfgcXu5eXJrRzwtX9rG1ftPwQ2NMw3TraX
// private= 2TtUBFZNELwKB4oeo4heRs3mZezaU84ujaHdVgCrki1jHhqn3yQYTaP92BrgrxeadEsStVE2f4aPL9ETsCeRGZjq

// addr= 9agPnEwDkAnoPMt2TenpwnBUc6djyykwepBKxUwZiHQE
// private= 1yjeMrBKG3phTqnZh5dnmBxxds4UBC7jBEM8DyTvpmmTk6iQ56yXzotdXTzD1nhBHjhjLu6WSzkYd5B3XaJurb4
package main

import (
	"context"
	"fmt"
	"kto/rpcclient"
	"kto/rpcclient/message"

	"google.golang.org/grpc"
)

func manin() {
	client, err := newClient()
	if err != nil {
		rpcclient.Error("fail to new client:", err)
		return
	}
	ctx := context.Background()
	reqData := &message.ReqTokenCreate{
		From:   "Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai",
		To:     "Kto3tPtVvkoBAgxxAgBbd6HpR1jgcfNWsz7554S4ud1PkBD",
		Amount: 500001,
		Nonce:  1,
		Priv:   "5BWVgtMPPUPFuCHssYhXxFx2xVfQRTkB1EjKHKu1B1KxdVxD5cswDEdqiko3PjUPFPGfePoKxdfzHvv4YXRCYNp2",
		Symbol: "BTC",
		Total:  "21000000",
	}
	resp, _ := client.CreateToken(ctx, reqData)
	fmt.Println(resp)

}
func newClient() (message.GreeterClient, error) {
	conn, err := grpc.Dial("106.12.88.252:8546", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		rpcclient.Error("fail to dial:", err)
		return nil, err
	}
	clinet := message.NewGreeterClient(conn)
	return clinet, nil
}

// func main() {
// 	loadCfg()
// 	client, err := newClient()
// 	if err != nil {
// 		rpcclient.Error("fail to new client:", err)
// 		return
// 	}
// 	ctx := context.Background()

// 	reqData := &message.ReqTransaction{
// 		From:   config.AddrMap["0"].Addr,
// 		Priv:   config.AddrMap["0"].Private,
// 		Amount: uint64(600000),
// 	}
// 	for i := 0; i < 1; i++ {
// 		var toAddr string
// 		if i/2 == 0 {
// 			toAddr = config.AddrMap["1"].Addr
// 		} else {
// 			toAddr = config.AddrMap["2"].Addr
// 		}
// 		rpcclient.Debugf("from:%s,to%s", config.AddrMap["0"].Addr, toAddr)

// 		nonce, err := client.GetAddressNonceAt(ctx, &message.ReqNonce{Address: config.AddrMap["0"].Addr})
// 		if err != nil {
// 			rpcclient.Error("fail to get nonce:", err)
// 			return
// 		}
// 		fmt.Println("nonce =======", nonce)
// 		rpcclient.Debug("nonce:", nonce.Nonce)

// 		reqData.To = toAddr
// 		for j := 0; j < 1; j++ {
// 			reqData.Nonce = nonce.Nonce + uint64(j)
// 			respData, err := client.SendTransaction(ctx, reqData)
// 			if err != nil {
// 				rpcclient.Errorf("fasttoother:%d,send tx error:%v", reqData.Nonce, err)
// 				return
// 			}
// 			rpcclient.Debug(respData.Hash)
// 		}

// 		// time.Sleep(8 * time.Second)
// 	}
// 	return
// }

// var config struct {
// 	AddrMap map[string]struct {
// 		Addr    string
// 		Private string
// 	}
// }

// func loadCfg() {
// 	_, err := toml.DecodeFile("./output/rpctest.toml", &config)
// 	if err != nil {
// 		rpcclient.Error(err)
// 		os.Exit(-1)
// 	}
// 	rpcclient.Debug(config)
// 	rpcclient.Debug(config.AddrMap["0"].Addr)
// }

// func fastToOther(ctx context.Context, client message.GreeterClient) error {

// 	reqData := &message.ReqTransaction{
// 		From:   config.AddrMap["0"].Addr,
// 		Priv:   config.AddrMap["0"].Private,
// 		Amount: uint64(500000),
// 	}
// 	for i := 0; i < 10; i++ {
// 		var toAddr string
// 		if i/2 == 0 {
// 			toAddr = config.AddrMap["1"].Addr
// 		} else {
// 			toAddr = config.AddrMap["2"].Addr
// 		}
// 		rpcclient.Debugf("from:%s,to%s", config.AddrMap["0"].Addr, toAddr)

// 		nonce, err := client.GetAddressNonceAt(ctx, &message.ReqNonce{Address: config.AddrMap["0"].Addr})
// 		if err != nil {
// 			rpcclient.Error("fail to get nonce:", err)
// 			return err
// 		}
// 		rpcclient.Debug("nonce:", nonce.Nonce)

// 		reqData.To = toAddr
// 		reqData.Nonce = nonce.Nonce
// 		respData, err := client.SendTransaction(ctx, reqData)
// 		if err != nil {
// 			rpcclient.Errorf("fasttoother:%d,send tx error:%v", reqData.Nonce, err)
// 			return err
// 		}
// 		rpcclient.Debug(respData.Hash)
// 		time.Sleep(8 * time.Second)
// 	}
// 	return nil
// }

// func sendEachOther(ctx context.Context, client message.GreeterClient) {
// 	var i int
// 	var fromAddr, toAddr string
// 	for {
// 		fromkey := strconv.Itoa(i % 3)
// 		tokey := strconv.Itoa((i + 1) % 3)
// 		rpcclient.Debugf("fromkey:%s,tokey:%s", fromkey, tokey)
// 		fromAddr = config.AddrMap[fromkey].Addr
// 		toAddr = config.AddrMap[tokey].Addr
// 		priv := config.AddrMap[fromkey].Private
// 		nonce, err := client.GetAddressNonceAt(ctx, &message.ReqNonce{Address: fromAddr})
// 		if err != nil {
// 			rpcclient.Error("fail to get nonce:", err)
// 			return
// 		}
// 		rpcclient.Debugf("fromkey:%s,nonce:%d\n", fromkey, nonce.Nonce)
// 		reqdata := &message.ReqTransaction{
// 			From:   fromAddr,
// 			To:     toAddr,
// 			Amount: uint64(500000),
// 			Nonce:  nonce.Nonce,
// 			Priv:   priv,
// 		}
// 		respdata, err := client.SendTransaction(ctx, reqdata)
// 		if err != nil {
// 			rpcclient.Errorf("snedEachother:%d,err:%v\n", reqdata.Nonce, err)
// 			return
// 		}
// 		rpcclient.Debug(respdata)
// 		i++
// 		time.Sleep(8 * time.Second)
// 	}
// }

// // func getBlockByNum(ctx context.Context, client message.GreeterClient) {
// // 	for {
// // 		time.Sleep(20 * time.Second)
// // 		respMaxNum, err := client.GetMaxBlockNumber(ctx, &message.ReqMaxBlockNumber{})
// // 		if err != nil {
// // 			rpcclient.Error("fail to get max num:", err)
// // 			continue
// // 		}

// // 		reqBlockByNum := &message.ReqBlock{Height: respMaxNum.MaxNumber}
// // 		respBlock, err := client.GetBlockbyNum(ctx, reqBlockByNum)
// // 		if err != nil {
// // 			rpcclient.Error("fail to get blcok by num:", err)
// // 			continue
// // 		}
// // 		rpcclient.Debug("number:", string(respBlock.Block))
// // 	}

// // }

// // func getBlockByHash(ctx context.Context, hash string, client message.GreeterClient) {
// // 	respBlock, err := client.GetBlockbyHash(ctx, &message.ReqBlock{Hash: hash})
// // 	if err != nil {
// // 		rpcclient.Error("fail to get block by hash:", err)
// // 		return
// // 	}
// // 	rpcclient.Debug("hash:", string(respBlock.Block))
// // }

// func keyPairTest() {
// 	pubbytes, privbytes, err := until.Generprivkey()
// 	if err != nil {
// 		rpcclient.Error(err)
// 		return
// 	}
// 	privkey := until.Encode(privbytes)
// 	pubkey := until.PubtoAddr(pubbytes)

// 	privBytes2 := until.Decode(privkey)
// 	// addr := "Kto" + until.Encode(privBytes[32:])
// 	addr2 := until.PubtoAddr(privBytes2[32:])
// 	if addr2 == pubkey {
// 		fmt.Println("success")
// 	}
// }

// func getBalance(ctx context.Context, client message.GreeterClient) {
// 	respBalance, err := client.GetBalance(ctx, &message.ReqBalance{
// 		Address: config.AddrMap["0"].Addr,
// 	})
// 	if err != nil {
// 		rpcclient.Error(err)
// 		return
// 	}
// 	respFrozenAssets, err := client.GetFrozenAssets(ctx, &message.ReqFrozenAssets{
// 		Addr: config.AddrMap["0"].Addr,
// 	})
// 	if err != nil {
// 		rpcclient.Error("fail to get frozen assets:", err)
// 		return
// 	}
// 	rpcclient.Debugf("before,balance:%d,forzenasssets:%d\n", respBalance.Balnce, respFrozenAssets.FrozenAssets)

// 	respLockBalance, err := client.SetLockBalance(ctx, &message.ReqLockBalance{
// 		Address: config.AddrMap["0"].Addr,
// 		Amount:  uint64(1000000),
// 	})
// 	rpcclient.Debug("lock:", respLockBalance.Status)

// 	respBalance, err = client.GetBalance(ctx, &message.ReqBalance{
// 		Address: config.AddrMap["0"].Addr,
// 	})
// 	if err != nil {
// 		rpcclient.Error(err)
// 		return
// 	}
// 	respFrozenAssets, err = client.GetFrozenAssets(ctx, &message.ReqFrozenAssets{
// 		Addr: config.AddrMap["0"].Addr,
// 	})
// 	if err != nil {
// 		rpcclient.Error("fail to get frozen assets:", err)
// 		return
// 	}
// 	rpcclient.Debugf("after,balance:%d,forzenasssets:%d\n", respBalance.Balnce, respFrozenAssets.FrozenAssets)
// }
