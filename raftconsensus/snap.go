package raftconsensus

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"path/filepath"

	pioutil "github.com/coreos/etcd/pkg/ioutil"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/snap/snappb"
)

const snapSuffix = ".snap"

var (
	crcTable         = crc32.MakeTable(crc32.Castagnoli)
	errNoSnapshot    = errors.New("snap: no available snapshot")
	errEmptySnapshot = errors.New("snap: empty snapshot")
	errCRCMismatch   = errors.New("snap: crc mismatch")
)

type snapshotter struct {
	dir string
}

func newSnapshotter(dir string) *snapshotter {
	return &snapshotter{dir: dir}
}

func (s *snapshotter) saveSnap(snapshot raftpb.Snapshot) error {
	if raft.IsEmptySnap(snapshot) {
		return nil
	}
	fname := fmt.Sprintf("%016x-%016x%s", snapshot.Metadata.Term, snapshot.Metadata.Index, snapSuffix)
	//b := pbutil.MustMarshal(snapshot)
	b, err := snapshot.Marshal()
	if err != nil {
		return err
	}
	crc := crc32.Update(0, crcTable, b)
	snap := snappb.Snapshot{Crc: crc, Data: b}
	d, err := snap.Marshal()
	if err != nil {
		return err
	}

	spath := filepath.Join(s.dir, fname)
	err = pioutil.WriteAndSyncFile(spath, d, 0666)
	if err != nil {
		return err
	}
	return nil
}
func (s *snapshotter) load(term, index uint64) (*raftpb.Snapshot, error) {
	fname := fmt.Sprintf("%016x-%016x%s", term, index, snapSuffix)
	fpath := filepath.Join(s.dir, fname)
	snap, err := read(fpath)
	if err != nil {
		return nil, err
	}
	return snap, nil

}

func read(fpath string) (*raftpb.Snapshot, error) {
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errEmptySnapshot
	}
	var serializedSnap snappb.Snapshot
	if err = serializedSnap.Unmarshal(b); err != nil {
		return nil, err
	}
	if len(serializedSnap.Data) == 0 || serializedSnap.Crc == 0 {
		return nil, errCRCMismatch
	}

	crc := crc32.Update(0, crcTable, serializedSnap.Data)
	if crc != serializedSnap.Crc {
		return nil, errCRCMismatch
	}
	var snap raftpb.Snapshot
	if err = snap.Unmarshal(serializedSnap.Data); err != nil {
		return nil, err
	}
	return &snap, nil

}
