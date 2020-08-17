package transaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kto/types"
	"kto/until/miscellaneous"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

type Transaction struct {
	Nonce       uint64        `json:"nonce"`
	BlockNumber uint64        `json:"blocknumber"`
	Amount      uint64        `json:"amount"`
	From        types.Address `json:"from"`
	To          types.Address `json:"to"`
	Hash        []byte        `json:"hash"`
	Signature   []byte        `json:"signature"`
	Time        int64         `json:"time"`
	Fee         uint64        `json:"fee"`
	Root        []byte        `json:"root"`
	Script      string        `json:"script"`
	Ord         Order         `json:"ord"`
	Errmsg      string        `json:"errmsg"`

	//Index     uint64
}

type Order struct {
	Id         []byte        `json:"id"`
	Address    types.Address `json:"address"`
	Price      uint64        `json:"price"`
	Hash       []byte        `json:"hash"`
	Ciphertext []byte        `json:" ciphertext"`
	Signature  []byte        `json:"signature"`
	Tradename  string        `json:"tradename"`
	Region     string        `json:"region"`
}

type Account struct {
	Nonce   uint64 `json:"nonce"`
	Balance uint64 `json:"balance"`
}

const Lenthaddr = 44

type Address [Lenthaddr]byte

func New() *Transaction {
	tx := &Transaction{}
	return tx

}

func (tx *Transaction) Tobytes() []byte {
	tx_byte, err := json.Marshal(tx)
	if err != nil {
		fmt.Println("Tobytes:", err)
		return nil
	}
	return tx_byte
}

func (tx *Transaction) GetTime() int64 {

	return int64(tx.Time)
}
func (tx *Transaction) GetNonce() int64 {

	return int64(tx.Nonce)
}
func (tx *Transaction) NewTransaction(nonce, amount uint64, from types.Address, to types.Address, script string) *Transaction {
	//创建交易

	t := time.Now().Unix()
	time := miscellaneous.E64func(uint64(t))

	tx = &Transaction{
		Nonce:     nonce,
		Amount:    amount,
		From:      from,
		To:        to,
		Hash:      nil,
		Signature: nil,
		Time:      t,
		Fee:       0,
		Root:      nil,
		Script:    script,
		Errmsg:    "",
	}

	var hashBytes []byte
	nonce_byte := miscellaneous.E64func(nonce)
	amountBytes := miscellaneous.E64func(amount)
	hashBytes = append(hashBytes, nonce_byte...)
	hashBytes = append(hashBytes, amountBytes...)
	hashBytes = append(hashBytes, from[:]...)
	hashBytes = append(hashBytes, to[:]...)
	hashBytes = append(hashBytes, time...)
	hash := sha3.Sum256(hashBytes)
	tx.Hash = hash[:]
	return tx
}

func (tx *Transaction) Newtoken(nonce, amount, fee uint64, from types.Address, to types.Address, script string) *Transaction {
	//创建交易

	t := time.Now().Unix()
	time := miscellaneous.E64func(uint64(t))

	tx = &Transaction{
		Nonce:     nonce,
		Amount:    amount,
		From:      from,
		To:        to,
		Hash:      nil,
		Signature: nil,
		Time:      t,
		Fee:       fee,
		Root:      nil,
		Script:    script,
		Errmsg:    "",
	}

	var hashBytes []byte
	nonce_byte := miscellaneous.E64func(nonce)
	amountBytes := miscellaneous.E64func(amount)
	hashBytes = append(hashBytes, nonce_byte...)
	hashBytes = append(hashBytes, amountBytes...)
	hashBytes = append(hashBytes, from[:]...)
	hashBytes = append(hashBytes, to[:]...)
	hashBytes = append(hashBytes, time...)
	hash := sha3.Sum256(hashBytes)
	tx.Hash = hash[:]
	return tx
}

// ID ,address,price,hash,sign,cip
func (tx *Transaction) NewOrder(id, cip []byte, address types.Address, price uint64) *Transaction {

	order := Order{}
	var hashBytes []byte
	priceBytes := miscellaneous.E64func(price)
	hashBytes = append(hashBytes, priceBytes...)
	hashBytes = append(hashBytes, id...)
	hashBytes = append(hashBytes, cip...)
	hashBytes = append(hashBytes, address[:]...)
	hash := sha3.Sum256(hashBytes)
	order.Hash = hash[:]
	order.Id = id
	order.Ciphertext = cip
	order.Address = address
	order.Price = price
	tx.Ord = order
	return tx
}
func (tx *Transaction) SignOrder(priv []byte) *Transaction {

	pri := ed25519.PrivateKey(priv)
	signatures := ed25519.Sign(pri, tx.Ord.Hash)
	tx.Ord.Signature = signatures
	return tx
}

//对交易签名
func (tx *Transaction) SignTx(priv []byte) *Transaction {

	pri := ed25519.PrivateKey(priv)
	if len(pri) != 64 {
		return nil
	}
	signatures := ed25519.Sign(pri, tx.Hash)
	tx.Signature = signatures
	return tx
}

//发送交易
func (tx *Transaction) SendTransaction() (string, error) {
	//发送post请求交易到服务端
	url := `http://127.0.0.1:12345/ReceTransaction?`

	req := &fasthttp.Request{}
	req.SetRequestURI(url)

	tx_byte, _ := json.Marshal(tx)

	requestBody := tx_byte
	req.SetBody(requestBody)

	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	resp := &fasthttp.Response{}

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		fmt.Println("请求失败:", err.Error())
		return "", err
	}

	b := resp.Body()
	var respdata struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(b, &respdata); err != nil {
		fmt.Println(err)
	}
	fmt.Println("hash==", respdata.Hash)
	return respdata.Hash, nil
}

func (tx *Transaction) HashTransaction() {
	fromBytes := tx.From[:]
	toBytes := tx.To[:]
	nonceBytes := miscellaneous.E64func(tx.Nonce)
	amountBytes := miscellaneous.E64func(tx.Amount)
	timeBytes := miscellaneous.E64func(uint64(tx.Time))
	txBytes := bytes.Join([][]byte{nonceBytes, amountBytes, fromBytes, toBytes, timeBytes}, []byte{})
	hash := sha3.Sum256(txBytes)
	tx.Hash = hash[:]
}

func (tx *Transaction) TrimmedCopy() *Transaction {
	txCopy := &Transaction{
		Nonce:  tx.Nonce,
		Amount: tx.Amount,
		From:   tx.From,
		To:     tx.To,
		Time:   tx.Time,
	}
	return txCopy
}

func (tx *Transaction) Sgin(privateKey []byte) {
	signatures := ed25519.Sign(ed25519.PrivateKey(privateKey), tx.Hash)
	tx.Signature = signatures
}

func (tx *Transaction) Verify() bool {
	txCopy := tx.TrimmedCopy()
	txCopy.HashTransaction()
	publicKey := AddressToPublicKey(string(tx.From[:]))
	return ed25519.Verify(publicKey, txCopy.Hash, tx.Signature)
}
