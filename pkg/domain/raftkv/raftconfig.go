package raftkv

import "go.etcd.io/etcd/raft"

var RaftConfig *raft.Config

func SetUpRaftConfig(id uint64) {
	RaftConfig = &raft.Config{
		ID: uint64(id),
		ElectionTick: 10,
		HeartbeatTick: 1,
		Storage: raftStorage,
		MaxSizePerMsg: 1024 * 1024,
		MaxInflightMsgs: 256,
		MaxUncommittedEntriesSize: 1 << 30,
	}
}