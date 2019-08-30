package raftkv

import (
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"go.etcd.io/etcd/wal"
	"go.etcd.io/etcd/wal/walpb"
	"go.uber.org/zap"
	"os"
)

var (
	walog    *wal.WAL
	walIndex uint64
)

func walExist() bool {
	walDir := fmt.Sprintf("wal-%d", raftConfig.ID)
	return wal.Exist(walDir)
}
func openWal() {
	walDir := fmt.Sprintf("wal-%d", raftConfig.ID)
	if !wal.Exist(walDir) {
		if err := os.Mkdir(walDir, 0750); err != nil {
			log.Error(fmt.Sprintf("raftexample: cannot create dir for wal (%v)", err))
		}
		w, err := wal.Create(zap.NewExample(), walDir, nil)
		if err != nil {
			log.Error(err.Error())
		} else {
			_ = w.Close()
		}
	}
	walSnap := walpb.Snapshot{}
	raftSnap = loadSnapShot()
	if raftSnap != nil {
		walSnap.Index, walSnap.Term = raftSnap.Metadata.Index, raftSnap.Metadata.Term
		_ = raftStorage.ApplySnapshot(*raftSnap)
	}
	var err error
	log.Info(fmt.Sprintf("Loading wal at term:%d,index:%d", walSnap.Term, walSnap.Index))
	walog, err = wal.Open(zap.NewExample(), walDir, walSnap)
	if err != nil {
		log.Fatal(fmt.Sprintf("Wal open error:%s", err.Error()))
	}
}

func prepareWal() {
	log.Info("Replaying wal")
	openWal()
	//read wal
	_, state, entries, err := walog.ReadAll()
	if err != nil {
		log.Fatal(fmt.Sprintf("Fail to read wal : %s", err.Error()))
	}
	//if raftSnap != nil {
	//	_ = raftStorage.ApplySnapshot(*raftSnap)
	//}
	_ = raftStorage.SetHardState(state)
	_ = raftStorage.Append(entries)
	if len(entries) > 0 {
		walIndex = entries[len(entries)-1].Index
	} else {
		// hell confusing here
		CommitC <- nil //trigger snapshot
	}
}
