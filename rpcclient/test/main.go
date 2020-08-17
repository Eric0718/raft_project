package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"kto/rpcclient"
	"kto/rpcclient/message"
	"kto/transaction"
	"kto/types"
	"kto/until"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"google.golang.org/grpc"
)

const (
	coinbaseAddr = "Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai"
	coinbasePriv = "5BWVgtMPPUPFuCHssYhXxFx2xVfQRTkB1EjKHKu1B1KxdVxD5cswDEdqiko3PjUPFPGfePoKxdfzHvv4YXRCYNp2"
)

type Client struct {
	client      message.GreeterClient
	ctx         context.Context
	addressList []userInfo
}

func main() {
	addr := flag.String("a", "106.12.9.134:8544", "host and post")
	num := flag.Int("n", 100, "每秒的次数")
	sleeptime := flag.Uint("t", 2, "sleep time")
	symbol := flag.String("symbol", "otk", "代币名称")
	addrFileName := flag.String("addrfile", "", "address json file")
	flag.Parse()
	fmt.Printf("\taddress\t%s\n", *addr)
	fmt.Printf("\tnum\t%d\n", *num)
	fmt.Printf("\tsymbol\t%s\n", *symbol)
	fmt.Printf("\taddrfile\t%s\n", *addrFileName)
	fmt.Printf("\tsleeptime\t%d\n", *sleeptime)

	runtime.GOMAXPROCS(runtime.NumCPU())
	//loadCfg()
	client, err := newClient(*addr, *addrFileName)
	if err != nil {
		rpcclient.Error("fail to new client:", err)
		return
	}
	/*离线交易*/
	err = client.sendSignedTx()
	fmt.Println(err)

	//client.createPriv()
	/* 代币 */
	// balance := client.getTokenBalance("Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai", *symbol)
	// fmt.Printf("balance:%d\n", balance)

	// tokenTotal := client.allTokenBalance(*symbol)
	// fmt.Printf("%s的总量为:%d\n", *symbol, tokenTotal)

	// _, _, err = client.mintToken(coinbaseAddr, coinbasePriv, *symbol, 100000, 1000000000000000, 500001)
	// if err != nil {
	// 	rpcclient.Error("client.mintToken failed", err)
	// 	return
	// }

	// time.Sleep(2 * time.Second)

	// err = client.tokenCycleTransaction(*symbol, 500001, *num)
	// if err != nil {
	// 	rpcclient.Error("client.tokenCycleTransaction failed", err)
	// 	return
	// }

	//测试大量的qtj转账
	// var total int
	// if *num <= 100 {
	// 	total = *num * 5
	// } else {
	// 	total = *num * 2
	// }
	// fmt.Printf("地址：%s,账户个数:%d,次数:%d,睡眠时间:%d\n", *addr, total, *num, *sleeptime)
	// befor := time.Now()
	// client.fastTx(total)
	// fmt.Println(time.Now().Sub(befor))

	// client.sendto100(*num, *sleeptime)

	//获取所有地址的总余额
	//client.getbalance()

	// if err := QTJTEST(ctx, client); err != nil {
	// 	rpcclient.Error(err)
	// }
	//KtoFT6eP7iLRTBiW7vjzUfvrj8VdxvniVnUnPW4nvzcXicM",
	//"priv":"2W3SQHGqGrc5ikBXSKxHm4JbHEWFNFWy6TRpmXPEYpH4CMw41ktjPRfWERChMmJ75hgjpN6Q57iM922Tq2Xwez4m"
	// ===============================================================
	// reqData := &message.ReqTransaction{
	// 	From:   "Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai",
	// 	To:     "KtoFT6eP7iLRTBiW7vjzUfvrj8VdxvniVnUnPW4nvzcXicM",
	// 	Amount: 100000000000000000,
	// 	Nonce:  nonce.Nonce,
	// 	Priv:   "5BWVgtMPPUPFuCHssYhXxFx2xVfQRTkB1EjKHKu1B1KxdVxD5cswDEdqiko3PjUPFPGfePoKxdfzHvv4YXRCYNp2",
	// }
	// respData, err := client.SendTransaction(ctx, reqData)
	// if err != nil {
	// 	fmt.Println("err")
	// 	return
	// }

	// sendEachOther(ctx, client)
	// keyPairTest()
	//getBalance(ctx, client)
	//largeSendTx(ctx, client)
	//createKerPair(ctx, client)
}

var config struct {
	AddrMap map[string]struct {
		Addr    string
		Private string
	}
}

func loadCfg() {
	_, err := toml.DecodeFile("./output/rpctest.toml", &config)
	if err != nil {
		rpcclient.Error(err)
		os.Exit(-1)
	}
	rpcclient.Debug(config)
	rpcclient.Debug(config.AddrMap["0"].Addr)
}

type Info struct {
	address string
	priv    string
}
type userInfo struct {
	Addr  string `json:"addr"`
	Priv  string `json:"priv"`
	Nonce uint64
}

// 创建地址测试
func (c *Client) createPriv() {
	fmt.Println("--------------createPriv--------------")
	var i int
	for {
		i++
		addrResp, err := c.client.CreateAddr(context.Background(), &message.ReqCreateAddr{})
		if err != nil {
			fmt.Printf("c.client.CreateAddr failed i:%d,error:%v\n", i, err)
			return
		}
		fmt.Printf("\ri:%d,address:%s", i, addrResp.Address)
		time.Sleep(100 * time.Millisecond)
	}
}
func (c *Client) sendSignedTx() error {
	wallet := transaction.NewWallet()
	fmt.Printf("to:%s,len:%d\n", wallet.Address, len(wallet.Address))
	nonceResp, err := c.client.GetAddressNonceAt(context.Background(), &message.ReqNonce{Address: coinbaseAddr})
	if err != nil {
		return err
	}

	tx := transaction.Transaction{
		From:   types.BytesToAddress([]byte(coinbaseAddr)),
		To:     types.BytesToAddress([]byte(wallet.Address)),
		Amount: uint64(10000000000),
		Nonce:  nonceResp.Nonce,
		Time:   time.Now().Unix(),
	}
	tx.HashTransaction()
	privateKey := until.Decode(coinbasePriv)
	tx.Sgin(privateKey)
	req := &message.ReqSignedTransaction{
		From:      coinbaseAddr,
		To:        wallet.Address,
		Amount:    tx.Amount,
		Nonce:     tx.Nonce,
		Time:      tx.Time,
		Hash:      tx.Hash,
		Signature: tx.Signature,
	}
	signedTxResp, err := c.client.SendSignedTransaction(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Println(signedTxResp.Hash)
	return nil
}

// 获取TokenAddress.josn中代币额
func (c *Client) allTokenBalance(symbol string) uint64 {
	fmt.Println("-------------allTokenBalance--------------")
	if c.addressList == nil || len(c.addressList) == 0 {
		rpcclient.Error("address list error")
		return 0
	}
	var allBlance uint64
	for _, address := range c.addressList {
		balance := c.getTokenBalance(address.Addr, symbol)
		if balance == 0 {
			fmt.Printf("address:%s,balance:%d", address.Addr, balance)
		}
		allBlance += balance
	}
	return allBlance
}

//获取token余额
func (c *Client) getTokenBalance(addr, symbol string) uint64 {
	fmt.Println("----------------getTokenBalance-------------")
	balanceResp, err := c.client.GetBalanceToken(context.Background(),
		&message.ReqTokenBalance{Address: addr, Symbol: symbol})
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return balanceResp.Balnce
}

// 创建代币
func (c *Client) mintToken(from, priv, symbol string, total, amount, fee uint64) (addr, topriv string, err error) {
	addressList := createAddress(1)
	var req = &message.ReqTokenCreate{
		From:   from,
		Priv:   priv,
		To:     addressList[0].Addr,
		Amount: amount,
		Symbol: symbol,
		Total:  total,
		Fee:    fee,
	}

	nonceResp, err := c.client.GetAddressNonceAt(context.Background(), &message.ReqNonce{Address: req.From})
	if err != nil {
		return "", "", err
	}
	req.Nonce = nonceResp.Nonce

	hashResp, err := c.client.CreateContract(context.Background(), req)
	if err != nil {
		rpcclient.Error("c.client.CreateContract failed", err)
		return "", "", err
	}
	fmt.Printf("\r%s", hashResp.Hash)

	time.Sleep(1 * time.Second)
	req.Nonce++
	hashResp, err = c.client.MintToken(context.Background(), req)
	if err != nil {
		rpcclient.Error("c.client.MintToken failed", err)
		return "", "", err
	}
	fmt.Printf("\r%s", hashResp.Hash)

	return addressList[0].Addr, addressList[0].Priv, nil
}

func readAddress(filename string) (userList []userInfo) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil
	}

	err = json.Unmarshal(data, &userList)
	if err != nil {
		return nil
	}
	return userList
}

// 循环发送代币
func (c *Client) tokenCycleTransaction(symbol string, fee uint64, n int) (err error) {
	if c.addressList == nil {
		c.addressList = createAddress(n)
	}

	//首先先从铸币者手中向其他地址发送代币
	req := &message.ReqTokenTransaction{
		From:        coinbaseAddr,
		Priv:        coinbasePriv,
		Amount:      10000000000000, //一百块
		TokenAmount: 100,
		Symbol:      symbol,
		Fee:         fee,
	}

	nonceResp, err := c.client.GetAddressNonceAt(context.Background(), &message.ReqNonce{Address: req.From})
	if err != nil {
		rpcclient.Error("c.client.GetAddressNonceAt failed", err)
		return err
	}
	req.Nonce = nonceResp.Nonce

	for i, address := range c.addressList {
		req.To = address.Addr
		hashResp, err := c.client.SendToken(context.Background(), req)
		if err != nil {
			rpcclient.Errorf("c.client.SendToken failed i:%d error:%v", i, err)
			return err
		}
		fmt.Printf("\r%s", hashResp.Hash)
		req.Nonce++
		time.Sleep(500 * time.Millisecond)
	}
	c.addressList = append(c.addressList, userInfo{coinbaseAddr, coinbasePriv, req.Nonce})
	//写入文件
	jsbyte, _ := json.Marshal(&c.addressList)
	err = ioutil.WriteFile("TokenUsers.json", jsbyte, 0644)
	if err != nil {
		rpcclient.Error("ioutil.WriteFile failed", err)
		return
	}

	for {
		for i, address := range c.addressList {
			req = &message.ReqTokenTransaction{
				From:        address.Addr,
				Priv:        address.Priv,
				To:          c.addressList[uint64((i+1)%len(c.addressList))].Addr,
				Nonce:       address.Nonce,
				Amount:      500001,
				TokenAmount: 1,
				Symbol:      symbol,
				Fee:         fee,
			}
			hashResp, err := c.client.SendToken(context.Background(), req)
			if err != nil {
				rpcclient.Errorf("c.client.SendToken failed i:%d nonce:%d error:%v", i, req.Nonce, err)
				return err
			}
			fmt.Printf("\r%s", hashResp.Hash)
			c.addressList[i].Nonce++
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (c *Client) sendto100(num int, sleeptime uint) error {
	var userList []userInfo
	data, err := ioutil.ReadFile("userinfo.json")
	if err != nil {
		log.Fatal("read file failed:", err)
	}
	err = json.Unmarshal(data, &userList)
	if err != nil {
		log.Fatal("parse json failed:", err)
	}

	if len(userList) == 0 || len(userList)%num != 0 {
		panic("不是整数倍")
	}

START:
	var wg sync.WaitGroup
	var t int64
	for i := 0; i < len(userList); i += num {
		t++
		wg.Add(num)
		for j := 0; j < num; j++ {
			go func(i, j int) {
				var reqData = &message.ReqTransaction{}
				defer func() {
					wg.Done()
				}()

				reqData.From = userList[i+j].Addr
				reqData.Priv = userList[i+j].Priv

				nonceData, err := c.client.GetAddressNonceAt(c.ctx, &message.ReqNonce{Address: reqData.From})
				if err != nil {
					rpcclient.Debug("addr:", reqData.From)
					rpcclient.Error("get nonce failed:", err)
					return
				}
				reqData.Nonce = nonceData.Nonce
				reqData.Amount = 100000000000 //1块钱

				// balanceData, err := c.client.GetBalance(c.ctx, &message.ReqBalance{Address: reqData.From})
				// if err != nil {
				// 	rpcclient.Debug("addr:", reqData.From)
				// 	rpcclient.Error("get banlance failed:", err)
				// 	return
				// }
				// if balanceData.Balnce < reqData.Amount {
				// 	return
				// }

				for x := 0; x < 10; x++ {
					reqData.To = userList[(i+j+x+10)%len(userList)].Addr

					//加上order
					// reqData.Order = &message.Order{}
					// reqData.Order.Id = fmt.Sprintf("%032d", reqData.Nonce)
					// reqData.Order.Address = "KtoBA91mi8mFEKnmQyZ698tXa5i89mesyXb2KvwoqyyRx21"
					// reqData.Order.Price = 600000
					// reqData.Order.Ciphertext = hex.EncodeToString([]byte("Kto2LRDy2u84ty9N5p9d455YhDAqj1dxM1bJxZyRcT6aNmm"))

					// signord, err := c.client.SignOrd(c.ctx, &message.ReqSignOrd{Priv: "3aPiZgy3dCcPXG2ZYZiTzT5QD3HBFcYRK88NxSqunkZj522i8swhAh7fZnqRBPeTF1j4MnGXxrdkgamhQebV7ogy", Order: reqData.Order})
					// if err != nil {
					// 	rpcclient.Error("sign ord failed:", err)
					// 	return
					// } else {
					// 	reqData.Order.Hash = signord.Hash
					// 	reqData.Order.Signature = signord.Signature
					// }
					beforTime := time.Now()
					_, err = c.client.SendTransaction(c.ctx, reqData)
					if err != nil {
						rpcclient.Debugf("i+j:%d,i+j+x+1:%d", i+x, (i+j+x+10)%100)
						rpcclient.Debug("from addr:", reqData.From)
						rpcclient.Debug("to addr:", reqData.To)

						rpcclient.Errorf("nonce:%d,send tx error:%v\n", reqData.Nonce, err)
						return
					}

					interval := time.Now().Sub(beforTime)
					if interval < (100 * time.Millisecond) {
						time.Sleep(100*time.Millisecond - interval)
					}

					reqData.Nonce++
				}

			}(i, j)
		}
		wg.Wait()
		if num > 50 && t%4 == 0 {
			fmt.Println("sleep..............")
			time.Sleep(time.Duration(sleeptime) * time.Second)
		}
	}
	goto START
	return nil
}

func (c *Client) fristTx() error {
	reqData := &message.ReqTransaction{
		From:   "Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai",
		Priv:   "5BWVgtMPPUPFuCHssYhXxFx2xVfQRTkB1EjKHKu1B1KxdVxD5cswDEdqiko3PjUPFPGfePoKxdfzHvv4YXRCYNp2",
		Amount: 1000000000000000, //100块
		To:     "KtoGoyujGLnWttEfAwmofdTzeXG534nrSZy6UR74uZ5mcoy",
	}

	nonce, err := c.client.GetAddressNonceAt(c.ctx, &message.ReqNonce{Address: reqData.From})
	if err != nil {
		rpcclient.Error("fail to get nonce:", err)
		return err
	}
	reqData.Nonce = nonce.Nonce

	hash, err := c.client.SendTransaction(c.ctx, reqData)
	if err != nil {
		panic(err)
	}

	fmt.Println("hash:", hash.Hash)
	return nil
}

func (c *Client) dataOverflow() error {

	var count int64
	var list [3]Info
	list[0].address = "KtoGoyujGLnWttEfAwmofdTzeXG534nrSZy6UR74uZ5mcoy"
	list[0].priv = "4UqKNPRjdXQsja2Geafu6BtLzqxLcScfTSBfSVxtav6gEq4LA2raLpP5hVNgTqdHPZ1rR9N7LbkstbzYeQhvVAXq"
	list[1].address = "KtoC5gP1TLyUWbHRkp1gfpMrbdBawnqxQi3NdYtB31dgtJE"
	list[1].priv = "5JFcGkBXdhD1E6Kgdrk8XWHRzvAuwbnezzmxkLLa4bfHPyWfG3LUBJLhVGZz7Y9ZcTocXaeqBMzbZfdo9yWyxZhc"
	list[2].address = "Kto3tPtVvkoBAgxxAgBbd6HpR1jgcfNWsz7554S4ud1PkBD"
	list[2].priv = "3zT8gZ3qstBbHCpB6Gm7T3DL2qz6SW5g5HaoK8EMx5BgxNou6Sn4n5Hpc5gVLFGefEfEsiRwUGdcBRXyLmxzRR6T"

	//START:
	reqData := &message.ReqTransaction{
		From: list[count%3].address,
		Priv: list[count%3].priv,
	}

	nonce, err := c.client.GetAddressNonceAt(c.ctx, &message.ReqNonce{Address: reqData.From})
	if err != nil {
		rpcclient.Error("fail to get nonce:", err)
		return err
	}
	rpcclient.Debug("nonce:", nonce.Nonce)

	balance, err := c.client.GetBalance(c.ctx, &message.ReqBalance{Address: reqData.From})
	if err != nil {
		return err
	}
	rpcclient.Debug("balance:", balance.Balnce)

	var num uint64 = 10
	if balance.Balnce > 100 {

		reqData.Amount = (balance.Balnce + 1000000000000) / num
		reqData.Nonce = nonce.Nonce

		rpcclient.Debug("amount:", reqData.Amount)
		rpcclient.Debug("count:", count)

		for i := 0; i < int(num); i++ {
			if i%2 == 0 {
				reqData.To = list[(count+1)%3].address
			} else {
				reqData.To = list[(count+2)%3].address
			}

			respData, err := c.client.SendTransaction(c.ctx, reqData)
			if err != nil {
				rpcclient.Debug("i:", i)
				rpcclient.Errorf("dataoverflow:%d,send tx error:%v\n", reqData.Nonce, err)
				break
			}

			rpcclient.Debug(respData.Hash)
			reqData.Nonce++
		}
	} else {
		rpcclient.Debug("balance:", balance.Balnce)
	}

	count++
	time.Sleep(5 * time.Second)
	//goto START

	return nil
}

func createAddress(num int) []userInfo {
	var data []userInfo

	for i := 0; i < num; i++ {
		pubBytes, privBytes, err := until.Generprivkey()
		if err != nil {
			log.Fatal(err)
		}
		var userData userInfo
		userData.Addr = until.PubtoAddr(pubBytes)
		userData.Priv = until.Encode(privBytes)
		userData.Nonce = 1
		if len(userData.Addr) != 47 {
			i--
			continue
		}

		data = append(data, userData)
	}

	// 写入本地文件
	// jsbyte, _ := json.Marshal(&data)
	// err := ioutil.WriteFile("userinfo.json", jsbyte, 0644)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	return data
}

// fastTx 因冲突删掉函数体
func (c *Client) fastTx(num int) {}

func newClient(host, addrFileName string) (*Client, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		rpcclient.Error("fail to dial:", err)
		return nil, err
	}

	var clinet = &Client{
		client:      message.NewGreeterClient(conn),
		ctx:         context.Background(),
		addressList: readAddress(addrFileName),
	}

	return clinet, nil
}
