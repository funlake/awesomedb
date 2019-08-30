package raftkv

import (
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"go.etcd.io/etcd/etcdserver/api/snap"
	"go.etcd.io/etcd/pkg/fileutil"
	"go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
	"go.etcd.io/etcd/wal/walpb"
	"go.uber.org/zap"
	"os"
)

var (
	snapshot *snap.Snapshotter
	raftSnap *raftpb.Snapshot
	defaultSnapshotCount uint64 = 10000
)

func prepareSnap() {
	snapDir := fmt.Sprintf("snap-%d",raftConfig.ID)
	if !fileutil.Exist(snapDir) {
		if err := os.Mkdir(snapDir,0750); err != nil {
			log.Fatal(fmt.Sprintf("Create snap dir error : %s",err.Error()))
		}
	}
	snapshot = snap.New(zap.NewExample(),snapDir)
}
func loadSnapShot() *raftpb.Snapshot{
	ss, err := snapshot.Load()
	if err != nil && err != snap.ErrNoSnapshot{
		log.Fatal(fmt.Sprintf("Snap shot load errr:%s",err.Error()))
	}
	return ss
}

func saveSnapShot(snap raftpb.Snapshot) error {
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term: snap.Metadata.Term,
	}
	if err := walog.SaveSnapshot(walSnap);err != nil {
		return err
	}
	if err := snapshot.SaveSnap(snap); err != nil {
		return err
	}
	return walog.ReleaseLockTo(snap.Metadata.Index)
}

func publishSnapShot(snap raftpb.Snapshot) {
	if raft.IsEmptySnap(snap) {
		return
	}
	CommitC <- nil // trigger kvstore to load snapshot
	log.Info(fmt.Sprintf("Start to publish snapshot at index %d",snap.Metadata.Index))
	if snap.Metadata.Index < appliedIndex {
		log.Fatal(fmt.Sprintf("Error publishing snapshot while snap > applied : %d > %d",snap.Metadata.Index,appliedIndex))
	}
	defer log.Info(fmt.Sprintf("Finish publishing snapshot at index %d",snap.Metadata.Index))
	confState 		= snap.Metadata.ConfState
	snapshotIndex 	= snap.Metadata.Index
	appliedIndex    = snap.Metadata.Index
}