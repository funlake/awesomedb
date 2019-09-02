package dbraft

import (
	"bytes"
	"encoding/json"
	sm "github.com/lni/dragonboat/v3/statemachine"
	"github.com/pkg/errors"
	"hash/fnv"
	"io"
	"io/ioutil"
	"sync"
)

func NewMemoryStateMachine(clusterId uint64, nodeId uint64) sm.IStateMachine {
	return &memoryStateMachine{
		kvstore:   make(map[string]string),
		clusterId: clusterId,
		nodeId:    nodeId,
	}
}

type memoryStateMachine struct {
	sync.Mutex
	kvstore   map[string]string
	clusterId uint64
	nodeId    uint64
}

func (msm *memoryStateMachine) Lookup(query interface{}) (interface{}, error) {
	msm.Lock()
	defer msm.Unlock()
	val, ok := msm.kvstore[query.(string)]
	if !ok {
		return "", errors.New("Not Found")
	}
	return val, nil
}
func (msm *memoryStateMachine) Update(data []byte) (sm.Result, error) {
	cmd := bytes.SplitN(data, []byte(" "), 2)
	key, val := string(cmd[0]), string(cmd[1])
	msm.Lock()
	defer msm.Unlock()
	msm.kvstore[key] = val
	return sm.Result{Value: uint64(len(data))}, nil
}
func (msm *memoryStateMachine) SaveSnapshot(w io.Writer,
	fc sm.ISnapshotFileCollection, done <-chan struct{}) error {
	snap, err := json.Marshal(msm.kvstore)
	if err != nil {
		return err
	}
	_, err = w.Write(snap)
	return err
}
func (msm *memoryStateMachine) RecoverFromSnapshot(r io.Reader,
	files []sm.SnapshotFile,
	done <-chan struct{}) error {
	var re map[string]string
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &re)
	if err != nil {
		msm.Lock()
		msm.kvstore = re
		msm.Unlock()
	}
	return err
}
func (msm *memoryStateMachine) Close() error { return nil }

func (msm *memoryStateMachine) GetHash() (uint64, error) {
	// the only state we have is that Count variable. that uint64 value pretty much
	// represents the state of this IStateMachine
	//return s.Count, nil
	snap, _ := json.Marshal(msm.kvstore)
	h := fnv.New64a()
	_, _ = h.Write(snap)
	return h.Sum64(), nil
}
