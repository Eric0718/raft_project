package raftconsensus

import (
	"kto/server"

	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/rafthttp"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal"
)

type raftNode struct {
	//proposeC    chan string
	//confChangeC chan raftpb.ConfChange // proposed cluster config changes
	errorC    chan<- error // errors from raft session
	clusterid uint64
	id        int    // client ID for raft session
	addr      string // node address
	addrPort  string
	peers     []string // raft peer URLs
	idPeer    map[int]string
	join      bool   // node is joining an existing cluster
	waldir    string // path to WAL directory
	snapdir   string // path to snapshot directory
	DS        string
	Cm        string
	QTJ       string
	blockData []byte //block data
	srv       *server.Server

	snapshotter    *snap.Snapshotter
	snapshot       *raftpb.Snapshot
	snapData       []byte
	snapCount      uint64
	confState      *raftpb.ConfState
	snapshotIndex  uint64
	appliedIndex   uint64
	curHeight      uint64
	lastTerm       uint64
	committedIndex uint64
	lastHeight     uint64
	// raft backing for the commit/error channel
	node        raft.Node
	raftStorage *raft.MemoryStorage
	wal         *wal.WAL
	nodeState   raft.Status //node status

	transport *rafthttp.Transport
	stopc     chan struct{} // signals proposal channel closed
	httpstopc chan struct{} // signals http server to shutdown
	httpdonec chan struct{} // signals http server shutdown complete

	logFile     string
	logSaveDays int
	logLevel    int
	logSaveMode int
	logFileSize int64
}

type cfgInfo struct {
	id        int
	addr      string
	peer      string
	peers     []string
	join      bool
	waldir    string
	snapdir   string
	raftport  int64
	Ds        string
	Cm        string
	QTJ       string
	snapCount int64

	logFile     string
	logSaveDays int
	logLevel    int
	logSaveMode int
	logFileSize int64
}

//using to write cluster config json file
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

type Request struct {
	RequestType NodeChangeType
	RequestId   int
	RequestPeer string
}

type NodeInfo struct {
	Id     int    `json:id`
	Peer   string `json:peer`
	Cli    *clusterInfo
	Result string
}
type NodeManage struct {
	rn *raftNode
}

type ReSSnaprpc struct {
	Done              bool
	Data              []byte
	Term              uint64
	NextSnapshotIndex uint64
}
type ReqSnaprpc struct {
	MaxTerm           uint64
	Term              uint64
	NextSnapshotIndex uint64
}

type NodeChangeType int32

const (
	RaftAddNode     NodeChangeType = 0
	RaftRemoveNode  NodeChangeType = 1
	RaftGetNodeInfo NodeChangeType = 2
)
