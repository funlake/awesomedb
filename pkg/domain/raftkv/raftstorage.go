package raftkv

import "go.etcd.io/etcd/raft"

var raftStorage = raft.NewMemoryStorage()
