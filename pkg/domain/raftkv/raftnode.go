package raftkv

import (
	"context"
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"go.etcd.io/etcd/pkg/types"
	"go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
	"strings"
	"time"
)

type RaftNode struct {
	Node raft.Node
}
func (rc *RaftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rc.Node.Step(ctx, m)
}
func (rc *RaftNode) IsIDRemoved(id uint64) bool                           { return false }
func (rc *RaftNode) ReportUnreachable(id uint64)                          {}
func (rc *RaftNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {}
var (
	raftConfig    *raft.Config
	raftPeers     []raft.Peer
	peersHosts    []string
	raftNode      raft.Node
	CommitC       = make(chan *string)
	ProposeC      = make(chan string)
	confChangeC   = make(chan raftpb.ConfChange)
	confState     raftpb.ConfState
	appliedIndex  uint64
	snapshotIndex uint64
)
func StartRaftNode (id int,join bool,peers string) {
	raftConfig =  &raft.Config{
		ID: uint64(id),
		ElectionTick: 10,
		HeartbeatTick: 1,
		Storage: raftStorage,
		MaxSizePerMsg: 1024 * 1024,
		MaxInflightMsgs: 256,
		MaxUncommittedEntriesSize: 1 << 30,
	}
	wexist := walExist()
	SetPeers(peers)
	//先设置snap
	prepareSnap()
	//设置本地缓存,可从snap目录恢复数据
	go prepareLocalStorage()
	//设置wal(write ahead log)
	//1.所有条目先进入wal,达成最终共识后,打入存储层
	//2.定时打snapshot到硬盘,wal只保留设置大小的条目，防止过大
	// 2.1 - 启动时恢复snapshot到raft storage
	//3.启动时从wal目录恢复数据到raft storage,包括有
	// 3.1 - HardState
	//在 HardState 里面，保存着该 Raft 节点最后一次保存的 term 信息，之前 vote 的哪一个节点，以及已经 commit 的 log index
	// 3.2 - Entries
	//把wal里记录的最后一次entries条目重新恢复到raft storage
	prepareWal()
	//启动raft node
	if wexist {
		raftNode = raft.RestartNode(raftConfig)
	} else {
		if join {
			raftNode = raft.StartNode(raftConfig, nil)
		} else {
			raftNode = raft.StartNode(raftConfig, GetPeers())
		}
	}
	//设置raft node 传输监听
	//transport看起来就是个简单的http transport
	//为何这个地方不用grpc,raft节点之间的通讯难道都用的http传输?
	startTransport(&RaftNode{ Node:raftNode})
	//监听状态机变更
	//1.更新本地缓存值
	//2.提交变更propose
	//3.监听confState(所有节点的状态信息),做相应的节点添加,删除
	//4.本地内存存储适时打snapshot
	go handleRaft()
}

func SetPeers(peers string) {
	peersHosts = strings.Split(peers,",")
	for pi := range peersHosts {
		raftPeers = append(raftPeers,raft.Peer{ID:uint64(pi+1)})
	}
}
func GetPeers() []raft.Peer {
	return raftPeers
}
func closeRaft(){
	close(ProposeC)
	close(confChangeC)
	close(transportStopC)
	close(CommitC)
	transport.Stop()
	raftNode.Stop()
}
func handleRaft() {
	snap,err := raftStorage.Snapshot()
	if err != nil {
		panic(err)
	}
	confState = snap.Metadata.ConfState
	appliedIndex = snap.Metadata.Index
	snapshotIndex = snap.Metadata.Index
	handlePropose()
	handleConfigChange()
	handleStateMachine()
	walog.Close()
}
func handlePropose(){
	go func() {
		for {
			select {
			case prop, ok := <-ProposeC:
				if !ok {
					closeRaft()
					return
				}
				// blocks until accepted by raft state machine
				_ = raftNode.Propose(context.TODO(), []byte(prop))
			}
		}
	}()
}
func handleConfigChange() {
	go func() {
		confChangeCount := uint64(0)
		for {
			select {
			case cc, ok := <- confChangeC:
				if !ok {
					closeRaft()
					return
				}
				confChangeCount++
				cc.ID = confChangeCount
				_ = raftNode.ProposeConfChange(context.TODO(), cc)
			}
		}
	}()
}

func handleStateMachine() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <- ticker.C:
			raftNode.Tick()
		case pd := <- raftNode.Ready():
			_ = walog.Save(pd.HardState, pd.Entries)
			if !raft.IsEmptySnap(pd.Snapshot) {
				_ = saveSnapShot(pd.Snapshot)
				_ = raftStorage.ApplySnapshot(pd.Snapshot)
				publishSnapShot(pd.Snapshot)
			}
			_ = raftStorage.Append(pd.Entries)
			transport.Send(pd.Messages)
			if ok := handleEntries(entriesToApply(pd.CommittedEntries));!ok {
				closeRaft()
				return
			}
			//todo: 本地存储与raft存储的同步
			//...
			MemoryStorage.makeSnapshot()
			raftNode.Advance()
		case <- transport.ErrorC:
			closeRaft()
			return
		}
	}
}

func handleEntries(entries []raftpb.Entry) bool {
	for idx := range entries {
		switch entries[idx].Type {
		case raftpb.EntryNormal:
			if len(entries[idx].Data) == 0 {
				break
			}
			s := string(entries[idx].Data)
			//log.Info(fmt.Sprintf("recieve msg %s",s))
			select {
			case CommitC <- &s:
			}
		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			_ = cc.Unmarshal(entries[idx].Data)
			confState = *raftNode.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					transport.AddPeer(types.ID(cc.NodeID),[]string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == raftConfig.ID {
					log.Warn(fmt.Sprintf("Node %d has been removed from other member",raftConfig.ID))
					return false
				}
				transport.RemovePeer(types.ID(cc.NodeID))
			}
		}
		appliedIndex = entries[idx].Index
		//why the fuck need stuff like this?
		if appliedIndex == walIndex {
			select {
			case CommitC <- nil :

			}
		}
	}
	return true
}
func entriesToApply (commitedEntries []raftpb.Entry) (es []raftpb.Entry) {
	if len(commitedEntries) == 0 {
		return commitedEntries
	}
	start := commitedEntries[0].Index
	if start > appliedIndex + 1 {
		log.Fatal(fmt.Sprintf("First index should not bigger than appliedIndex,%d>%d",start,appliedIndex))
	}
	if (appliedIndex-start+1) < uint64(len(commitedEntries)) {
		es = commitedEntries[appliedIndex-start+1:]
	}
	return es
}