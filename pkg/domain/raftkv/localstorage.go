package raftkv

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"go.etcd.io/etcd/etcdserver/api/snap"
	"sync"
)
type kv struct {
	Key string
	Val string
}
var MemoryStorage *LocalStorage
type LocalStorage struct {
	kvStore map[string] string
	mu sync.Mutex
	//why it must be a pointer?
	snapshotter *snap.Snapshotter
}
func prepareLocalStorage() {
	MemoryStorage = &LocalStorage{
		kvStore: make(map[string] string),
	}
	//MemoryStorage.handleCommitC()
	go MemoryStorage.handleCommitC()
}
func (s *LocalStorage) handleCommitC(){
	for msg := range CommitC {
		log.Info(fmt.Sprintf("msg from commitc : %#v",msg))
		if msg == nil {
			//recover from snapshot
			ss := loadSnapShot()
			if ss != nil {
				if err := s.recoverFromSnapshot(ss.Data); err != nil {
					log.Fatal(fmt.Sprintf("Recover from snapshot fails %s", err.Error()))
				}
			}
			continue
		}
		var msgKv kv
		doc := gob.NewDecoder(bytes.NewBufferString(*msg))
		if err := doc.Decode(&msgKv);err != nil {
			log.Fatal(fmt.Sprintf("Decode commits fail : %s",err.Error()))
		}
		s.mu.Lock()
		s.kvStore[msgKv.Key] = msgKv.Val
		s.mu.Unlock()
	}
}
func (s *LocalStorage) Propose(key,val string) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(kv{key,val});err != nil {
		log.Fatal(fmt.Sprintf("Gob encode error : %s",err.Error()))
	}
	ProposeC <- buf.String()
}
func (s *LocalStorage) Find(key string) (string,bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val,ok := s.kvStore[key]
	return val,ok
}
func (s *LocalStorage) getSnapshot() ([]byte,error){
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Marshal(s.kvStore)
}
func (s *LocalStorage) makeSnapshot() {
	if appliedIndex-snapshotIndex+ 1 < defaultSnapshotCount {
		return
	}
	log.Info("Begin snapshot local storage")
	data,err := s.getSnapshot()
	if err != nil {
		log.Fatal(err.Error())
	}
	snap,err := raftStorage.CreateSnapshot(appliedIndex,&confState,data)
	if err != nil {
		log.Fatal(fmt.Sprintf("Raft storage create snap shot error:%s",err.Error()))
	}
	if err := saveSnapShot(snap); err != nil {
		log.Fatal(fmt.Sprintf("Save snap shot error:%s",err.Error()))
	}
	compactIndex := uint64(0)
	if appliedIndex > defaultSnapshotCount {
		compactIndex = appliedIndex - defaultSnapshotCount
	}
	if err := raftStorage.Compact(compactIndex);err != nil {
		log.Fatal(fmt.Sprintf("Compact error at %d",compactIndex))
	}
	log.Info(fmt.Sprintf("Compact successfully at %d",compactIndex))
	snapshotIndex = appliedIndex
}
func (s *LocalStorage) recoverFromSnapshot(snapshot []byte) error {
	var data map[string]string
	log.Info("Start to recover snapshot")
	if err := json.Unmarshal(snapshot,&data);err != nil {
		return err
	}
	s.mu.Lock()
	s.kvStore = data
	s.mu.Unlock()
	return nil
}