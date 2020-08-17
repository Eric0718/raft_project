package main

/*************************************
Usage:
**First ,must do "./noderpc_client [本机ip:port] get/-G/GET " to get cluster info and find which one is leader.
Then do next step 1 or 2 on leader.
1. Add node:
	./noderpc_client [本机ip:port] [-A/ADD/add] [id] [ip:port]
2. Delete node
	./noderpc_client [本机ip:port] [-D/del/delete/DELETE] [id]

*************************************/
import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"

	"github.com/howeyc/gopass"
	"github.com/spf13/viper"
)

const PASSWD = "kto123456"

type Request struct {
	RequestType NodeChangeType
	RequestId   int
	RequestPeer string
}

type clusterInfo struct {
	ClusterID    uint64   `json:"ClusterID"`
	Peers        []string `json:"Peers"`
	LeaderID     int      `json:"LeaderID"`
	LeaderPeer   string   `json:"LeaderPeer"`
	NodeID       int      `json:"NodeID"`
	NodePeer     string   `json:"NodePeer"`
	TotalNodeNum int      `json:"TotalNodeNum"`
	MaxNodeId    int      `json:"MaxID"`
}

type NodeInfo struct {
	Id         int    `json:id`
	Peer       string `json:peer`
	LeaderID   int    `leaderID`
	LeaderPeer string `leaderPeer`
	Cli        *clusterInfo
	Result     string
}

type NodeChangeType int32

const (
	RaftAddNode     NodeChangeType = 0
	RaftRemoveNode  NodeChangeType = 1
	RaftGetNodeInfo NodeChangeType = 2
)

func handleRequst(args []string, conn *rpc.Client) {
	aLen := len(args)
	if aLen < 1 {
		fmt.Println("[handleRequst] : Wrong parameter! Try ./noderpc_client -h/-H/--help")
		return
	}
	switch args[0] {
	case "-a", "-A", "add":
		identityCheck()

		if aLen != 3 {
			log.Fatalln("[ADD]:Wrong parameter！Try ./noderpc_client -h/-H/--help")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalln("[ADD]Wrong paramete: %v. Try ./noderpc_client -h/-H/--help: %v", err)
		}

		req := Request{
			RequestType: RaftAddNode,
			RequestId:   id,
			RequestPeer: args[2],
		}
		res := NodeInfo{}
		err = conn.Call("NodeManage.HandleNode", &req, &res)
		if err != nil {
			log.Fatalln("NodeManage addNode error: ", err)
		}
		if res.Result != "" {
			fmt.Println(res.Result)
		}
	case "-d", "-D", "delete":
		identityCheck()
		if aLen != 2 {
			log.Fatalln("[DEL]:Wrong parameter! Try ./noderpc_client -h/-H/--help")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalln("[DEL]Wrong parameter! Try ./noderpc_client -h/-H/--help: %v", err)
		}

		req := Request{
			RequestType: RaftRemoveNode,
			RequestId:   id,
		}
		res := NodeInfo{Result: ""}
		fmt.Println("Request to delete node", req.RequestId)
		err = conn.Call("NodeManage.HandleNode", &req, &res)
		if err != nil {
			log.Fatalln("NodeManage delete Node error: ", err)
		}
		if res.Result != "" {
			fmt.Println(res.Result)
		}
	case "-g", "-G", "get":
		identityCheck()

		req := Request{RequestType: RaftGetNodeInfo}
		res := NodeInfo{Result: ""}

		err := conn.Call("NodeManage.HandleNode", &req, &res)
		if err != nil {
			log.Fatalln("NodeManage get Node info error: ", err)
		}
		fmt.Printf("Leader: %v\n", res.Cli.LeaderID)
		fmt.Printf("Leader peer: %v\n", res.Cli.LeaderPeer)
		fmt.Printf("Self id: %v\n", res.Cli.NodeID)
		fmt.Printf("Self peer: %v\n", res.Cli.NodePeer)
		fmt.Printf("Cluster total node number: %v\n", res.Cli.TotalNodeNum)
		fmt.Printf("Cluster max node id: %v\n", res.Cli.MaxNodeId)
		fmt.Println("Cluster peers: ", res.Cli.Peers)
	case "-h", "-H", "--help":
		fmt.Println("Usage:")
		fmt.Println("Get cluster info: ./noderpc_client -g/-G/get")
		fmt.Println("Add a node: ./noderpc_client -a/-A/add/ADD [id] [ip:port]")
		fmt.Println("Remove a node: ./noderpc_client -d/-D/delete [id] [ip:port]")
	default:
		log.Fatalln("[default]:Wrong parameter! Try ./noderpc_client -h/-H/--help")
	}
	return
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Wrong parameter! Try ./noderpc_client -h/-H/--help")
		return
	}

	vp := viper.New()
	vp.SetConfigName("nodecfg") //把json文件换成yaml文件，只需要配置文件名 (不带后缀)即可
	vp.AddConfigPath("./conf")  //添加配置文件所在的路径
	vp.SetConfigType("yaml")    //设置配置文件类型
	err := vp.ReadInConfig()
	if err != nil {
		fmt.Printf("config file error: %s\n", err)
		os.Exit(1)
	}
	ipport := vp.GetString("peer")

	conn, err := rpc.DialHTTP("tcp", ipport)
	if err != nil {
		log.Fatalln("dailing error: ", err)
	}

	handleRequst(os.Args[1:], conn)
}

func identityCheck() {

	fmt.Printf("Please enter your password: ")
	pwd, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Wrong input:", err)
	}
	//fmt.Scanln(&passwd)
	if PASSWD != string(pwd) {
		log.Fatalln("Wrong password,please try again!")
	}
}
