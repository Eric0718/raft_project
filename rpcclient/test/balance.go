package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kto/rpcclient/message"
	"log"
)

func (c *Client) getbalance() {
	bytes, err := ioutil.ReadFile("userinfo.json")
	if err != nil {
		log.Panic(err)
	}
	var userList []userInfo

	err = json.Unmarshal(bytes, &userList)
	if err != nil {
		log.Panic(err)
	}
	frist, err := c.client.GetBalance(c.ctx, &message.ReqBalance{Address: "Kto9sFhbjDdjEHvcdH6n9dtQws1m4ptsAWAy7DhqGdrUFai"})
	if err != nil {
		log.Fatalf("get frist balance error:%v", err)
	}
	fmt.Printf("首地址:%d\n", frist.Balnce)
	var amount uint64
	for i, user := range userList {
		resp, err := c.client.GetBalance(c.ctx, &message.ReqBalance{Address: user.Addr})
		if err != nil {
			log.Fatalf("总数：%d,当前:%d，error:%v", len(userList), i, err)
		}
		amount += resp.Balnce
	}

	fmt.Println("总金额：", amount)
}
