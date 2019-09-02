package dbraft

import (
	"bufio"
	"context"
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/lni/goutils/syncutil"
	"os"
	"strings"
	"time"
)

const clusterId = 256

func StartMemoryRaftNode(id int, join bool, hosts string) {
	peers := make(map[uint64]string)
	// when joining a new node which is not an initial members, the peers map should
	// be empty.
	if !join {
		for idx, v := range strings.Split(hosts, ",") {
			// key is the NodeID, NodeID is not allowed to be 0
			// value is the raft address
			peers[uint64(idx+1)] = v
		}
	}
	var nodeAddr string
	// for simplicity, in this example program, addresses of all those 3 initial
	// raft members are hard coded. when address is not specified on the command
	// line, we assume the node being launched is an initial raft member.
	nodeAddr = peers[uint64(id)]

	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)

	rc := config.Config{
		NodeID:             uint64(id),
		ClusterID:          clusterId,
		ElectionRTT:        5,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    50,
		CompactionOverhead: 5,
	}
	walDir := fmt.Sprintf("mms-%d", id)
	rn := config.NodeHostConfig{
		WALDir:         walDir,
		NodeHostDir:    walDir,
		RTTMillisecond: 200,
		RaftAddress:    nodeAddr,
	}
	nh, err := dragonboat.NewNodeHost(rn)
	if err != nil {
		panic(err)
	}
	if err := nh.StartCluster(peers, join, NewMemoryStateMachine, rc); err != nil {
		log.Error(fmt.Sprintf("Start Cluster error:%s", err.Error()))
		os.Exit(-1)
	}
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				// no data will be lost/corrupted if nodehost.Stop() is not called
				nh.Stop()
				return
			}
			s = strings.Replace(s, "\n", "", -1)
			ch <- s
		}
	})
	raftStopper.RunWorker(func() {
		cs := nh.GetNoOPSession(clusterId)
		for {
			select {
			case cmd, ok := <-ch:
				if !ok {
					return
				}
				cmds := strings.Split(cmd, " ")
				if len(cmds) < 2 {
					log.Warn("UnSupport command")
					continue
				}
				if cmds[0] == "put" || cmds[1] == "set" {
					if len(cmds) < 3 {
						log.Warn("UnSupport command")
						continue
					}
					// input is a regular message need to be proposed
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					// make a proposal to update the IStateMachine instance
					_, err := nh.SyncPropose(ctx, cs, []byte(cmds[1]+" "+cmds[2]))
					cancel()
					if err != nil {
						log.Error(fmt.Sprintf("SyncPropose returned error %v\n", err))
					} else {
						log.Info(fmt.Sprintf("Set ok on node %d:%s->%s", rc.NodeID, cmds[1], cmds[2]))
					}
				} else if cmds[0] == "get" {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					result, err := nh.SyncRead(ctx, clusterId, cmds[1])
					cancel()
					if err != nil {
						log.Error(fmt.Sprintf("SyncRead returned error %v\n", err))
					} else {
						log.Info(fmt.Sprintf("Get key on node %d : %s ,result : %s", rc.NodeID, cmds[1], result))
					}
				} else {
					log.Info(fmt.Sprintf("query:%#v", cmds))
					log.Warn("UnSupport command")
					continue
				}
			}
		}
	})
	raftStopper.Wait()
}
