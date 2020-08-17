package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"kto/blockchain"
	"kto/p2p/node"
	"kto/raftconsensus"
	"kto/rpcclient"
	"kto/server"
	"kto/txpool"
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/viper"
)

type cfgInfo struct {
	id            int
	addr          string
	member        []string
	p2pport       int
	p2pip         string
	advertiseAddr string
	//	qtj     string
}

var cfg *cfgInfo

func init() {

	vp := viper.New()

	vp.SetConfigName("nodecfg") //把json文件换成yaml文件，只需要配置文件名 (不带后缀)即可
	vp.AddConfigPath("./conf")  //添加配置文件所在的路径
	vp.SetConfigType("yaml")    //设置配置文件类型
	err := vp.ReadInConfig()
	if err != nil {
		fmt.Printf("config file error: %s\n", err)
		os.Exit(1)
	}

	cg := &cfgInfo{
		id:      vp.GetInt("id"),
		addr:    vp.GetString("addr"),
		member:  vp.GetStringSlice("member"),
		p2pport: vp.GetInt("p2pport"),
		p2pip:   vp.GetString("p2pip"),
		//advertiseAddr: vp.GetString("advertiseAddr"),
		//qtj:     vp.GetString("qtj"),
	}
	cfg = cg
}
func main() {
	tp := txpool.New()
	bc := blockchain.New()
	fmt.Println(cfg.p2pport, "addr=", cfg.addr, "p2pip=", cfg.p2pip)
	n, err := node.New(cfg.p2pport, cfg.addr, cfg.p2pip, tp, bc)
	if err != nil {
		log.Fatal("node new is err:", err)
	}
	go n.Run()
	if cfg.member != nil {
		for _, member := range cfg.member {
			fmt.Println("cfg.member=========", member)
			fmt.Printf("main_join: %v\n", n.Join([]string{member}))
		}

	}
	s := server.New(bc, tp, n)
	go rpcclient.Run_rpc(bc, tp, n)
	go raftconsensus.RunNode(s)
	go s.Run()
	go func() {
		http.ListenAndServe(":8081", nil)
	}()
	for {
		time.Sleep(1 * time.Second)
	}
	//==============
	// for {
	// 	time.Sleep(1 * time.Second)
	// 	s.Chukuai()

	// }
	//==============

}
