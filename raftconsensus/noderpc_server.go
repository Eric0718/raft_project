package raftconsensus

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

func (nm *NodeManage) HandleNode(req *Request, res *NodeInfo) error {
	switch req.RequestType {
	case RaftAddNode:
		fmt.Println("Node rpc server adding node ", req.RequestId, req.RequestPeer)
		nm.rn.NodeManagement(req, res)
	case RaftRemoveNode:
		fmt.Println("Node rpc server deleting node ", req.RequestId)
		nm.rn.NodeManagement(req, res)
	case RaftGetNodeInfo:
		cls := nm.rn.GetClusterInfo()
		// data, err := json.Marshal(cls)
		// fmt.Println("\n\rNode rpc server GET DATA: ", data, "\n\r")
		// if err != nil {
		// 	fmt.Println("Marshal clusterInfo error:", err)
		// 	return err
		// }
		// res.ClusterInfo = data
		res.Cli = cls
	}

	return nil
}

func IsExit(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (nm *NodeManage) HandleSnap(req *ReqSnaprpc, res *ReSSnaprpc) error {
	term := req.Term
	for ; term < req.MaxTerm+1; term++ {

		fname := fmt.Sprintf("%016x-%016x%s", term, req.NextSnapshotIndex, snapSuffix)
		_, err := os.Stat("./log/snap/" + fname)
		if err == nil {
			snap := newSnapshotter("./log/snap")
			data, err := snap.load(term, req.NextSnapshotIndex)
			if err != nil {
				res.Done = false
				return err
			}
			if data == nil {
				res.Done = false
				return errors.New("load data is nil")
			}

			bData, err1 := json.Marshal(data)
			if err1 != nil {
				res.Done = false
				return err1
			}
			if len(bData) == 0 {
				res.Done = false
				return errors.New("data is 0")
			}
			res.Data = bData
			res.Done = true
			res.Term = term
			res.NextSnapshotIndex = req.NextSnapshotIndex

			return nil
		}
	}
	res.Done = false
	res.Term = term
	res.NextSnapshotIndex = req.NextSnapshotIndex
	return errors.New(fmt.Sprintf("Can not find requested (%016x-%016x)-%016x.snap file.", req.Term, req.MaxTerm, req.NextSnapshotIndex))
}

func (nm *NodeManage) ResquestSnapData(address string, req *ReqSnaprpc) error {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var res *ReSSnaprpc
	err = client.Call("NodeManage.HandleSnap", req, res)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
func (nm *NodeManage) RunNodeManage() {

	fmt.Println("runNodeManage=======...")
	err := rpc.Register(nm) // 注册rpc服务
	if err != nil {
		fmt.Println(err)
	}

	rpc.HandleHTTP() // 采用http协议作为rpc载体

	fmt.Println("net.Listen=======", nm.rn.addrPort)

	lis, err := net.Listen("tcp", nm.rn.addrPort)
	if err != nil {
		log.Fatalln("fatal error: ", err)
	}

	fmt.Println("http.Serve start=======")
	http.Serve(lis, nil)
}
