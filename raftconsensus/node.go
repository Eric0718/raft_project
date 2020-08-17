// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raftconsensus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"kto/block"
	"kto/server"
	"kto/txpool"
	"kto/until/logger"
	"net/http"
	"net/rpc"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/etcdserver/stats"
	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/coreos/etcd/pkg/types"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/rafthttp"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"
)

// A key-value stream backed by raft

// newRaftNode initiates a raft instance and returns a committed log entry
// channel and error channel. Proposals for log updates are sent over the
// provided the proposal channel. All log entries are replayed over the
// commit channel, followed by a nil message (to indicate the channel is
// current), then new log entries. To shutdown, close proposeC and read errorC.
func newRaftNode(cf *cfgInfo, s *server.Server) <-chan error {
	errorC := make(chan error)
	rn := &raftNode{
		clusterid:      0x1000,
		id:             cf.id,
		addr:           cf.addr,
		addrPort:       cf.peer,
		peers:          cf.peers,
		join:           cf.join,
		waldir:         cf.waldir,
		snapdir:        cf.snapdir,
		DS:             cf.Ds,
		Cm:             cf.Cm,
		QTJ:            cf.QTJ,
		srv:            s,
		snapCount:      uint64(cf.snapCount),
		errorC:         errorC,
		stopc:          make(chan struct{}),
		raftStorage:    raft.NewMemoryStorage(),
		idPeer:         make(map[int]string),
		logFile:        cf.logFile,
		logSaveDays:    cf.logSaveDays,
		logLevel:       cf.logLevel,
		logSaveMode:    cf.logSaveMode,
		logFileSize:    cf.logFileSize,
		appliedIndex:   1,
		lastHeight:     1,
		committedIndex: 0,
		snapshotIndex:  0,
		snapshot:       new(raftpb.Snapshot),
	}

	// 日志初始化
	logger.Config(rn.logFile, rn.logLevel)
	logger.SetSaveMode(rn.logSaveMode)
	if rn.logSaveMode == 3 && rn.logFileSize > 0 {
		logger.SetSaveSize(rn.logFileSize * 1024 * 1024)
	}
	logger.SetSaveDays(rn.logSaveDays)
	logger.Infof("start raft...\n")

	go rn.startNode()
	return errorC
}

func (rn *raftNode) committedEntriesToApply(ents []raftpb.Entry) (entrs []raftpb.Entry) {
	if len(ents) <= 0 {
		return nil
	}
	firstIdx := ents[0].Index
	if firstIdx > rn.committedIndex {
		logger.Fatalf("Fatal error:first index of committedEntries[%d] should <= progress appliedIndex[%d]+1\n", firstIdx, rn.committedIndex)
		panic("Fatal error:first index of committedEntries should <= progress appliedIndex+1")
	}
	if rn.committedIndex-firstIdx < uint64(len(ents)) {
		entrs = ents[rn.committedIndex-firstIdx:]
	}
	logger.Infof("committedEntriesToApply:[firstIdx = %v,rn.committedIndex+1 = %v],len(entrs) = %v.\n", firstIdx, rn.committedIndex+1, len(entrs))
	return entrs
}

func (rn *raftNode) EntriesToApply(ents []raftpb.Entry) []raftpb.Entry {

	if len(ents) <= 0 {
		return nil
	}
	for i, e := range ents {
		if len(e.Data) <= 0 {
			ents = append(ents[:i], ents[i+1:]...)
		}
	}

	logger.Infof("EntriesToApply: len(ents) = %v\n{commitentries: %v}.\n\r", len(ents), ents)
	return ents
}

// replayWAL replays WAL entries into the raft instance.
func (rn *raftNode) replayWAL() *wal.WAL {
	rn.createWal()
	err := rn.loadSnapshot()
	if err != nil {
		panic("Raft loadSnapshot error!")
	}
	return rn.readWAL()
}

// openWAL returns a WAL ready for reading.
func (rn *raftNode) readWAL() *wal.WAL {

	logger.Infof("replaying WAL of member %d", rn.id)
	walsnap := walpb.Snapshot{}
	if rn.snapshot != nil {
		walsnap.Index, walsnap.Term = rn.snapshot.Metadata.Index, rn.snapshot.Metadata.Term
	}
	logger.Infof("loading WAL at term %d and index %d", walsnap.Term, walsnap.Index)
	w, err := wal.Open(rn.waldir, walsnap)
	if err != nil {
		logger.Fatalf("raftconsensus: error loading wal (%v)", err)
	}

	_, st, ents, err := w.ReadAll()
	if err != nil {
		logger.Fatalf("raftconsensus: failed to read WAL (%v)", err)
	}

	if rn.snapshot != nil {
		rn.raftStorage.ApplySnapshot(*rn.snapshot)
		rn.confState = &rn.snapshot.Metadata.ConfState
		rn.snapshotIndex = rn.snapshot.Metadata.Index
		logger.Infof("Start: [snapshotIndex = %v,term = %v]\n", rn.snapshot.Metadata.Index, rn.snapshot.Metadata.Term)
	}
	rn.committedIndex = st.Commit
	rn.raftStorage.SetHardState(st)
	logger.Infof("Start: [committedIndex = %v,len(ents) = %v]\n", st.Commit, len(ents))

	// append to storage so raft starts at the right place in log
	rn.raftStorage.Append(ents)
	return w
}

// createWal replays WAL entries into the raft instance.
func (rn *raftNode) createWal() {

	if !wal.Exist(rn.waldir) {
		if err := os.MkdirAll(rn.waldir, 0750); err != nil {
			logger.Fatalf("raftconsensus error: cannot create dir for wal (%v)", err)
		}
		w, err := wal.Create(rn.waldir, nil)
		if err != nil {
			logger.Fatalf("raftconsensus error: create wal error (%v)", err)
		}
		w.Close()
	}

}

func (rn *raftNode) writeError(err error) {
	rn.stopHTTP()
	rn.errorC <- err
	close(rn.errorC)
	rn.node.Stop()
}

func (rn *raftNode) startNode() {
	if !fileutil.Exist(rn.snapdir) {
		if err := os.MkdirAll(rn.snapdir, 0750); err != nil {
			logger.Fatalf("raftconsensus error: cannot create dir for snapshot (%v)", err)
		}
	}
	rn.snapshotter = snap.New(rn.snapdir)
	oldwal := wal.Exist(rn.waldir)
	w := rn.replayWAL()
	if w == nil {
		logger.Fatalf("error: cannot create wal !!!")
	}
	rn.wal = w

	rpeers := make([]raft.Peer, len(rn.peers))
	for i := range rpeers {
		rpeers[i] = raft.Peer{ID: uint64(i + 1)}
		rn.idPeer[i+1] = rn.peers[i]
	}
	c := &raft.Config{
		ID:              uint64(rn.id),
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         rn.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
	}

	if oldwal {
		logger.Infof("raft->RestartNode=======")
		rn.node = raft.RestartNode(c)
	} else {
		startPeers := rpeers
		if rn.join {
			startPeers = nil
		}
		logger.Infof("raft->StartNode=======")
		rn.node = raft.StartNode(c, startPeers)
	}

	rn.transport = &rafthttp.Transport{

		ID:          types.ID(rn.id),
		ClusterID:   types.ID(rn.clusterid),
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(strconv.Itoa(rn.id)),
		ErrorC:      make(chan error),
		Raft:        rn,
	}
	rn.transport.Start()
	for i := range rn.peers {
		if i+1 != rn.id {
			rn.transport.AddPeer(types.ID(i+1), []string{rn.peers[i]})
			fmt.Println("Transport start: ID = ", i+1, "address = ", rn.peers[i])
		}
	}

	go rn.serveRaft()
	go rn.packBlock()
	go rn.handleChanel()

	nm := &NodeManage{rn: rn}
	go nm.RunNodeManage()
}

// stop closes http, closes all channels, and stops raft.
func (rn *raftNode) stop() {
	rn.stopHTTP()
	close(rn.errorC)
	rn.node.Stop()
}

func (rn *raftNode) stopHTTP() {
	rn.transport.Stop()
	close(rn.httpstopc)
	<-rn.httpdonec
}

func (rn *raftNode) handleChanel() {
	defer rn.wal.Close()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lh, err := rn.srv.Bc.Height()
	if err != nil {
		panic("handleChanel error: get last block height failed!")
	}
	rn.lastHeight = lh

	// event loop on raft state machine updates
	for {
		select {
		case <-ticker.C:
			rn.node.Tick()

		// store raft entries to wal, then publish over commit channel
		case rd := <-rn.node.Ready():
			//save snap:
			if !raft.IsEmptySnap(rd.Snapshot) {
				rn.recoverSnapshotData(rd.Snapshot)
				rn.raftStorage.ApplySnapshot(rd.Snapshot)
			}

			rn.commitEntries(rd.CommittedEntries)
			err := rn.wal.Save(rd.HardState, rd.Entries)
			if err != nil {
				logger.Errorf("Save wal: %v\n", err)
			}

			if rd.HardState.Commit != 0 {
				logger.Infof("Save wal successfully.\n")
			}
			if len(rd.Entries) != 0 {
				rn.raftStorage.Append(rd.Entries)
			}

			if !raft.IsEmptyHardState(rd.HardState) {
				//logger.Infof("SetHardState Info:  committedIndex,rd.HardState.Commit:[%v ,%v] \n", rn.committedIndex, rd.HardState.Commit)
				if rn.committedIndex < rd.HardState.Commit {
					logger.Infof("SetHardState Info: set rd.HardState.Commit to committedIndex:[%v ,%v] \n", rn.committedIndex, rd.HardState.Commit)
					rd.HardState.Commit = rn.committedIndex
				}
				rn.raftStorage.SetHardState(rd.HardState)
			}
			if len(rd.Messages) != 0 {
				rn.transport.Send(rd.Messages)
			}

			if rn.committedIndex > rn.snapshotIndex {
				if rn.committedIndex-rn.snapshotIndex >= rn.snapCount {
					offset := rn.committedIndex % rn.snapCount
					err := rn.maybeTriggerSnapshot(rn.committedIndex - offset)
					if err != nil {
						logger.Infof("Snapshot error: %v; new snap Index = %v,last snap Index = %v\n", err, rn.committedIndex, rn.snapshotIndex)
					}
				}
			}

			rn.node.Advance()

		case err := <-rn.transport.ErrorC:
			rn.writeError(err)
			return

		case <-rn.stopc:
			rn.stop()
			return
		}
	}
}

func (rn *raftNode) commitEntries(ents []raftpb.Entry) {
	if len(ents) <= 0 || ents == nil {
		return
	}
	for _, e := range ents {

		logger.Infof("commitEntries: e.Index=%v,last committedIndex = %v,e.term =%v.\n", e.Index, rn.committedIndex, e.Term)
		if e.Index <= rn.committedIndex {
			logger.Infof("Continue warning: log e.Index=%v already commited. Committed index = %v,term =%v.\n", e.Index, rn.committedIndex, e.Term)
			continue
		}
		if len(e.Data) > 0 {
			if e.Type == raftpb.EntryNormal {
				//commit block
				b := &block.Blocks{}
				err := json.Unmarshal(e.Data, b)
				if err != nil {
					logger.Infof("commitEntries Unmarshal warning: %v\n", err)
					continue
				}

				p := rn.srv.Gettxpool()
				p.Filter(*b)

				logger.Infof("Start: commit block b.height=%v，last block height = %v\n", b.Height, rn.lastHeight)
				if b.Height <= rn.lastHeight {
					logger.Infof("Commit warning: new block height [%v <= %v] last Height!\n", b.Height, rn.lastHeight)
					rn.committedIndex++
					continue
				}

				if b.Height-rn.lastHeight != 1 {
					logger.Errorf("Commit block fatal error:b.Height %v - rn.lastHeight %v = %v.\n", b.Height, rn.lastHeight, b.Height-rn.lastHeight)
					panic("Commit block error: b.Height - lastHeight >1")
				}

				if !rn.checkBlock(b) {
					logger.Infof("Follow checkBlock error: block height = %v !\n", b.Height)
					panic("CheckBlock ERROR!")
				}

				if !txpool.VerifyBlcok(*b, rn.srv.Bc) {
					logger.Infof("Follow verifyBlcok error: block height = %v !\n", b.Height)
					panic("VerifyBlcok ERROR!")
				}

				//start to commit
				err = rn.srv.CommitBlock(b, []byte(rn.addr))
				if err != nil {
					logger.Fatalf("Fatal error: commit block failed, height = %v,e.Index = %v,e.term = %v,txnum = %v\n", b.Height, e.Index, e.Term, len(b.Txs))
					fmt.Printf("Fatal error: commit block failed, Height =%v,e.index = %v,e.term = %v,txnum = %v\n", b.Height, e.Index, e.Term, len(b.Txs))
					panic("Commit block failed!")
				}

				//update lastHeight.
				rn.lastHeight = b.Height
				//logger.Infof("End: commited block height = %v,e.Index = %v,e.term = %v,txnum = %v\n", b.Height, e.Index, e.Term, len(b.Txs))
				fmt.Printf("End: commited Block Height =%v,e.index = %v,e.term = %v,txnum = %v\n", b.Height, e.Index, e.Term, len(b.Txs))
			}

			if e.Type == raftpb.EntryConfChange {
				var cc raftpb.ConfChange
				cc.Unmarshal(e.Data)
				rn.confState = rn.node.ApplyConfChange(cc)

				switch cc.Type {
				case raftpb.ConfChangeAddNode:
					if len(cc.Context) > 0 {
						fmt.Println("Add node into cluster :", cc.NodeID, string(cc.Context))
						rn.transport.AddRemote(types.ID(cc.NodeID), []string{string(cc.Context)})
						rn.transport.AddPeer(types.ID(cc.NodeID), []string{string(cc.Context)})
						rn.updateClusterInfo(true, types.ID(cc.NodeID), string(cc.Context))
					}
				case raftpb.ConfChangeRemoveNode:
					fmt.Println("Removed node from the cluster and Shutting down.", cc.NodeID)
					if rn.id != int(cc.NodeID) {
						rn.transport.RemovePeer(types.ID(cc.NodeID))
					}
					if rn.id == int(cc.NodeID) {
						rn.transport.Stop()
						rn.node.Stop()
					}
					rn.updateClusterInfo(false, types.ID(cc.NodeID), "")
				}
			}
		}

		// after commit, update committedIndex.
		rn.committedIndex = e.Index
	}
}

func (rn *raftNode) maybeTriggerSnapshot(appIndex uint64) error {
	logger.Infof("Snapshot info: appliedIndex = %v,last snapshotIndex = %v.\n", rn.committedIndex, rn.snapshotIndex)
	data, err := rn.getSnapData(appIndex + 1)
	if err != nil {
		logger.Infof("getSnapData error!")
		return err
	}
	snap, err := rn.raftStorage.CreateSnapshot(appIndex, rn.confState, data)
	if err != nil {
		logger.Infof("CreateSnapshot error!")
		return err
	}

	if err := rn.saveSnap(snap); err != nil {
		logger.Infof("saveSnap error!")
		return err
	}

	//Compact discards all log entries prior to compactIndex.
	if err := rn.raftStorage.Compact(appIndex); err != nil {
		logger.Infof("Compact error!")
		return err
	}
	logger.Infof("Compacted log at commited index %d", appIndex)
	//update rn.snapshotIndex
	rn.snapshotIndex = appIndex
	rn.snapshot = &snap
	logger.Infof("End: Finish snapshot at Index = %v\n", appIndex)
	return nil
}

func (rn *raftNode) unMarshalSnapData(snapData []byte) ([]raftpb.Entry, error) {

	if len(snapData) <= 0 || snapData == nil {
		logger.Errorf("snapData error: %v\n", snapData)
		return nil, errors.New("snapData error.")
	}

	var ents []raftpb.Entry
	err := json.Unmarshal(snapData, &ents)
	if err != nil {
		logger.Errorf("recoverSnapshotData error: %v\n", err)
		return nil, err
	}
	l := len(ents)
	if l <= 0 {
		logger.Infof("Warning: Snap entries length should > 0, return.\n")
		return nil, errors.New("Snap entries length should > 0, return.")
	}

	return ents, nil
}

func (rn *raftNode) recoverSnapshotData(snapshotToSave raftpb.Snapshot) {
	if raft.IsEmptySnap(snapshotToSave) {
		return
	}

	logger.Infof("publishing snapshot at index %d", rn.snapshotIndex)
	defer logger.Infof("finished publishing snapshot at index %d", rn.snapshotIndex)

	if snapshotToSave.Metadata.Index <= rn.committedIndex {
		logger.Infof("snapshot index [%d] should > progress committedIndex [%d]", snapshotToSave.Metadata.Index, rn.committedIndex)
		return
	}

	ents, err := rn.unMarshalSnapData(snapshotToSave.Data)
	if err != nil {
		logger.Errorf("recoverSnapshotData unMarshalSnapData: %v.", err)
		return
	}

	l := len(ents)
	if l <= 0 {
		logger.Errorf("Warning: Snap entries length should > 0, return.\n")
		return
	}

	if rn.snapshotIndex >= ents[l-1].Index {
		logger.Infof("Don't need to save snap.[rn.snapshotIndex %v >= %v ents[l-1].Index]\n", rn.snapshotIndex, ents[l-1].Index)
		return
	}

	//missing snap data. so get snap data from leader by rpc then commit.
	ci, _, err := rn.raftStorage.InitialState()
	if err != nil {
		logger.Errorf("commitEntries error: Get HardState failed , %v\n", err)
		return
	}

	rn.committedIndex = ci.Commit

	if rn.committedIndex < ents[0].Index && rn.snapshotIndex < ents[0].Index {
		var term uint64 = 1
		if rn.snapshot != nil {
			term = rn.snapshot.Metadata.Term
		}

		logger.Infof("recover missing SnapshotData: from last commited index = %v to ents[0].Index = %v,snapshotIndex = %v,term = %v,ents[l-1].Index = %v.\n", rn.committedIndex, ents[0].Index, rn.snapshotIndex, term, ents[l-1].Index)
		for nextReqSnapIndex := rn.snapshotIndex + rn.snapCount; (nextReqSnapIndex < snapshotToSave.Metadata.Index) && (rn.snapshotIndex%rn.snapCount == 0); {

			req := ReqSnaprpc{
				MaxTerm:           snapshotToSave.Metadata.Term,
				Term:              term,
				NextSnapshotIndex: nextReqSnapIndex,
			}
			res := ReSSnaprpc{}
			st := rn.node.Status()
			if st.Lead == 0 {
				//logger.Warnf("ResquestSnapData get ip error: leader id = 0. Waiting for electing leader,then restart.\n")
				continue
			}
			add, err := url.Parse(rn.idPeer[int(st.Lead)])
			if err != nil {
				//logger.Errorf("ResquestSnapData get ip error: %v\n", err)
				continue
			}
			leaderIP := add.Hostname()
			if len(leaderIP) <= 0 {
				//logger.Warnf("ResquestSnapData get ip error: len(leaderIP) <= 0.\n")
				continue
			}

			port := rn.addrPort
			p := strings.Split(port, ":")
			if len(p[1]) <= 0 {
				continue
			}
			addport := leaderIP + ":" + p[1]
			logger.Infof("ResquestSnapData gets ip:port = %v\n", addport)
			logger.Infof("recoverSnapshotData: Start to request snap data by rpc from term [%v-%v].\n", term, snapshotToSave.Metadata.Term)
			er := rn.ResquestSnapData(addport, &req, &res)
			if er != nil {
				logger.Errorf("ResquestSnapData error: %v.term = %v,snapshotindex = %v\n", er, res.Term, res.NextSnapshotIndex)
				panic("ResquestSnapData error!")
			}
			if res.Done {
				/***********************************/
				logger.Infof("recoverSnapshotData: Request snap data by rpc successfully.\n")
				data := res.Data
				snap := new(raftpb.Snapshot)
				err := json.Unmarshal(data, snap)
				if err != nil {
					logger.Errorf("Unmarshal error: %v\n", err)
					break
				}

				if raft.IsEmptySnap(*snap) {
					logger.Errorf("recoverSnapshotData saveSnap warning: Empty snap data.\n")
					break
				}

				//commit snap data
				if len(snap.Data) > 0 {
					en, err := rn.unMarshalSnapData(snap.Data)
					if err != nil {
						logger.Errorf("recover snap unMarshalSnapData error: %v!\n", err)
						break
					}
					entries, err := rn.snapEntriesToApply(rn.committedIndex+1, en)
					if err != nil {
						logger.Errorf("snapEntriesToApply error: %v!\n", err)
						break
					}
					logger.Infof("recoverSnapshotData: Start recover missing snap data from index %v to index %v,term = %v,snapshotindex = %v.\n", rn.committedIndex, entries[len(entries)-1].Index, res.Term, res.NextSnapshotIndex)
					rn.commitEntries(entries)
					//store snap data to disk
					if err := rn.saveSnap(*snap); err != nil {
						logger.Errorf("recoverSnapshotData saveSnap error1: %v!\n", err)
						break
					}

					rn.confState = &snap.Metadata.ConfState
					rn.snapshotIndex = snap.Metadata.Index
					rn.committedIndex = snap.Metadata.Index
					rn.snapshot = snap
					term = snap.Metadata.Term
					nextReqSnapIndex = rn.snapshotIndex + rn.snapCount
				}
			}
		}

		if rn.snapshotIndex+rn.snapCount != snapshotToSave.Metadata.Index {
			logger.Errorf("recoverSnapshotData error: recover missing snap data failed![snapIndex = %v,term = %v]\n", rn.snapshotIndex, term)
			panic("recoverSnapshotData error: recover missing snap data failed\n")
		}
		logger.Infof("recoverSnapshotData: Finish recover missing snap data.[snapIndex = %v,term = %v]\n", rn.snapshotIndex, term)
	}
	//no missing snap data, so recover data from newest sanp.
	logger.Infof("recoverSnapshotData: rn.snapshotIndex+rn.snapCount[%v + %v] [<=>] snapshotToSave.Metadata.Index [%v].\n", rn.snapshotIndex, rn.snapCount, snapshotToSave.Metadata.Index)
	if (rn.snapshotIndex+rn.snapCount == snapshotToSave.Metadata.Index) && (rn.snapshotIndex%rn.snapCount == 0) {
		//store snap data
		entries, err := rn.snapEntriesToApply(rn.committedIndex+1, ents)
		if err != nil {
			logger.Errorf("snapEntriesToApply error: %v!\n", err)
			return
		}
		logger.Infof("recoverSnapshotData: Start recover snap data.\n")
		rn.commitEntries(entries)
		//store snap data to disk
		if err := rn.saveSnap(snapshotToSave); err != nil {
			logger.Errorf("recoverSnapshotData saveSnap error2: %v!\n", err)
			panic("recoverSnapshotData saveSnap error2")
		}
		rn.confState = &snapshotToSave.Metadata.ConfState
		rn.snapshotIndex = snapshotToSave.Metadata.Index
		rn.committedIndex = snapshotToSave.Metadata.Index
		rn.snapshot = &snapshotToSave
		logger.Infof("recoverSnapshotData: Finish recover snap data.[snapIndex = %v,term = %v]\n", rn.snapshotIndex, snapshotToSave.Metadata.Term)

		// //discard log before committedIndex
		// fst, _ := rn.raftStorage.FirstIndex()
		// last, _ := rn.raftStorage.LastIndex()
		// if rn.committedIndex >= fst && rn.committedIndex <= last {
		// 	if err := rn.raftStorage.Compact(rn.committedIndex); err != nil {
		// 		logger.Infof("recoverSnapshotData compact error!")
		// 		return
		// 	}
		// 	logger.Infof("recoverSnapshotData compacted log at commited index %d", rn.committedIndex)
		// }
	}
}

func (rn *raftNode) snapEntriesToApply(ci uint64, ents []raftpb.Entry) ([]raftpb.Entry, error) {
	le := len(ents)
	if ci < ents[0].Index {
		logger.Infof("Commited index = %v ,ents[0].index = %v\n", ci, ents[0].Index)
		return nil, errors.New("Commited index should >= ents[0].index.")
	}
	if ci > ents[le-1].Index {
		return nil, errors.New("Already Commited, index should < ents[len - 1].index.")
	}
	return ents[ci-ents[0].Index:], nil
}

func (rn *raftNode) ResquestSnapData(address string, req *ReqSnaprpc, res *ReSSnaprpc) error {

	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = client.Call("NodeManage.HandleSnap", req, res)
	if err != nil {
		logger.Errorf("Handle snap error:", err)
		return err
	}
	return nil
}

func (rn *raftNode) loadSnapshot() error {
	snapshot, err := rn.snapshotter.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		logger.Fatalf("raftconsensus: error loading snapshot (%v)", err)
		return err
	}
	rn.snapshot = snapshot
	return nil
}

func (rn *raftNode) checkBlock(b *block.Blocks) bool {
	logger.Infof("====================Start checkBlock====================")
	if !rn.srv.Bc.Checkresults(b, []byte(rn.DS), []byte(rn.Cm), []byte(rn.QTJ)) {
		return false
	}
	logger.Infof("=====================End checkBlock====================")
	return true
}

func (rn *raftNode) packBlock() {
	for {
		time.Sleep(time.Second * 1)
		st := rn.node.Status()
		fmt.Printf("Leader ID:%v;  SelfNode Id:%v\n", st.Lead, rn.id)
		logger.Infof("Leader ID:%v;  SelfNode Id:%v\n", st.Lead, rn.id)

		if st.SoftState.RaftState == raft.StateLeader {

			li, err := rn.raftStorage.LastIndex()
			if err != nil {
				logger.Infof("Before package Warning: Get last Index failed !\n")
				continue
			}
			ci, _, err := rn.raftStorage.InitialState()
			if err != nil {
				logger.Infof("Warning: Get commit Index failed, %v.\n", err)
				continue
			}
			if li > ci.Commit {
				logger.Infof("Warning: Last Index [%v != %v] commited Index !\n", li, ci.Commit)
				continue
			}
			logger.Infof("Start to pack Block: Last log Index [%v] , Commited Index [%v].\n", li, ci.Commit)

			b, err := rn.srv.PackBlock([]byte(rn.addr), []byte(rn.DS), []byte(rn.Cm), []byte(rn.QTJ))
			if err != nil {
				logger.Infof("Leader: PackBlock failed,do it again.\n")
				continue
			}
			logger.Infof("Leader: Pack new Block height = %v successfully!\n", b.Height)
			b.Miner = []byte(rn.addr)
			b = rn.srv.Bc.Calculationresults(b)
			bs := block_change(b)

			pb, err := json.Marshal(bs)
			if err != nil {
				logger.Infof("Marshal error: package again.\n")
				continue
			}
			currentHeight, _ := rn.srv.Bc.Height()
			if bs.Height <= currentHeight {
				logger.Infof("Warning: New pack Height [%v <= %v] current db height.\n", bs.Height, currentHeight)
				continue
			}

			er := rn.node.Propose(context.TODO(), []byte(pb))
			if er != nil {
				logger.Errorf("Raft Propose: %v,continue...", er)
				continue
			}
			logger.Infof("End: Finish Propose new block height = %v.\n", bs.Height)
		}
	}
}

func (rn *raftNode) serveRaft() {
	url, err := url.Parse(rn.idPeer[rn.id])
	if err != nil {
		logger.Fatalf("raftconsensus error: Failed parsing URL (%v)", err)
	}
	ln, err := newStoppableListener(url.Host, rn.httpstopc)
	if err != nil {
		logger.Fatalf("raftconsensus error: Failed to listen rafthttp (%v)", err)
	}
	err = (&http.Server{Handler: rn.transport.Handler()}).Serve(ln)
	select {
	case <-rn.httpstopc:
	default:
		logger.Fatalf("raftconsensus error: Failed to serve rafthttp (%v)", err)
	}
	close(rn.httpdonec)
}

func (rn *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rn.node.Step(ctx, m)
}
func (rn *raftNode) IsIDRemoved(id uint64) bool                           { return false }
func (rn *raftNode) ReportUnreachable(id uint64)                          {}
func (rn *raftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}

func block_change(b *block.Block) *block.Blocks {
	var brs *block.Block_Res = &block.Block_Res{}
	var rst block.Blocks
	for addr, bal := range b.Txres {
		brs.Address = addr
		brs.Balance = bal
		rst.Res = append(rst.Res, *brs)
	}
	for _, mtx := range b.FirstTx {
		rst.FirstTx = append(rst.FirstTx, mtx)
	}
	var bs *block.Blocks = &block.Blocks{
		Height:        b.Height,
		PrevBlockHash: b.PrevBlockHash,
		Txs:           b.Txs,
		Root:          b.Root,
		Version:       b.Version,
		Timestamp:     b.Timestamp,
		Hash:          b.Hash,
		Miner:         b.Miner,
		Res:           rst.Res,
		FirstTx:       rst.FirstTx,
	}
	return bs
}

func (rn *raftNode) saveSnap(snap raftpb.Snapshot) error {
	// must save the snapshot index to the WAL before saving the
	// snapshot to maintain the invariant that we only Open the
	// wal at previously-saved snapshot indexes.
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}
	if err := rn.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	if err := rn.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	return rn.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (rn *raftNode) getSnapData(hi uint64) ([]byte, error) {
	lo := rn.snapshotIndex + 1
	ents, err := rn.raftStorage.Entries(lo, hi, 5*1024*1024*1024)
	if err != nil {
		logger.Infof("Get entries error from index %v to %v。", lo, hi)
		return nil, err
	}
	logger.Infof("getSnapData: Get entries from index %v to %v, get length = %v.\n", lo, hi, len(ents))
	data, err := json.Marshal(ents)
	if err != nil {
		logger.Infof("get Entries Marshal error!")
		return nil, err
	}
	return data, nil
}

func (rn *raftNode) updateClusterInfo(confchange bool, id types.ID, ps string) {
	fmt.Println("Update cluster info when config changed.")
	if confchange {
		if ps != "" {
			rn.peers = append(rn.peers, ps)

			if _, ok := rn.idPeer[int(id)]; !ok {
				rn.idPeer[int(id)] = ps
				fmt.Println("Clustermanage add node :", int(id), ps)
			}
		}
	} else {
		fmt.Println("Clustermanage delete node id =", int(id))
		for indx, p := range rn.peers {
			if p == rn.idPeer[int(id)] {
				rn.peers = append(rn.peers[:indx], rn.peers[indx+1:]...)
			}
		}
		if _, ok := rn.idPeer[int(id)]; ok {
			delete(rn.idPeer, int(id))
		}

	}

}

func (rn *raftNode) NodeManagement(req *Request, res *NodeInfo) {
	st := rn.node.Status()
	if st.SoftState.RaftState == raft.StateLeader {
		switch req.RequestType {
		case RaftAddNode:
			//如果已经存在，添加失败返回并带回结果信息。
			for k, v := range rn.idPeer {
				if k == req.RequestId {
					re := fmt.Sprintf("Failed: node %v is alrady exist!", req.RequestId)
					res.Result = re
					break
				}
				if v == req.RequestPeer {
					re := fmt.Sprintf("Node %v is alrady exist!", req.RequestPeer)
					res.Result = re
					break
				}
			}

			logger.Infof("Leader ADDs NODE ======", req.RequestId, "++++", req.RequestPeer)
			data, _ := json.Marshal(req.RequestPeer)
			cc := raftpb.ConfChange{
				Type:    0,
				NodeID:  uint64(req.RequestId),
				Context: data,
			}
			rn.node.ProposeConfChange(context.TODO(), cc)
		case RaftRemoveNode:
			//如果要删除的id不存在，删除失败返回并带回结果信息
			if _, ok := rn.idPeer[req.RequestId]; !ok {
				re := fmt.Sprintf("Failed: node id %v is not existing!!", req.RequestId)
				res.Result = re
				break
			}
			logger.Infof("Leader Removes NODE ======", req.RequestId)
			cc := raftpb.ConfChange{
				Type:   1,
				NodeID: uint64(req.RequestId),
			}
			rn.node.ProposeConfChange(context.TODO(), cc)
		}

	}

}

func (rn *raftNode) GetClusterInfo() *clusterInfo {
	maxid := 1
	for k, _ := range rn.idPeer {
		if maxid < k {
			maxid = k
		}
	}
	num := len(rn.peers)
	st := rn.node.Status()

	cls := &clusterInfo{
		ClusterID:    rn.clusterid,
		Peers:        rn.peers,
		LeaderID:     int(st.Lead),
		LeaderPeer:   rn.idPeer[int(st.Lead)],
		NodeID:       rn.id,
		NodePeer:     rn.idPeer[rn.id],
		TotalNodeNum: num,
		MaxNodeId:    maxid,
	}

	return cls
	// data, err := json.Marshal(cls)
	// if err != nil {
	// 	logger.Errorf("Marshal clusterInfo error:", err)
	// }
	// writeInfo(data)
}
