package raftkv

import (
	"fmt"
	"github.com/funlake/awesomedb/pkg/log"
	"go.etcd.io/etcd/etcdserver/api/rafthttp"
	stats "go.etcd.io/etcd/etcdserver/api/v2stats"
	"go.etcd.io/etcd/pkg/types"
	"go.uber.org/zap"
	"net/http"
	"net/url"
)
var (
	transport     *rafthttp.Transport
	transportStopC = make(chan struct{})
)
func startTransport(raftNode *RaftNode){
	transport = &rafthttp.Transport{
		Logger: zap.NewExample(),
		ID: types.ID(raftConfig.ID),
		ClusterID: 0x1000,
		Raft: raftNode,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(fmt.Sprintf("%d",raftConfig.ID)),
		ErrorC:      make(chan error),
	}
	_ = transport.Start()

	for _,r := range GetPeers() {
		if r.ID != raftConfig.ID {
			transport.AddPeer(types.ID(r.ID),[]string{peersHosts[r.ID - 1]})
		}
	}
	go serveTransport()
}

func serveTransport() {
	hostParse,err := url.Parse(peersHosts[transport.ID - 1])
	if err != nil {
		log.Fatal(fmt.Sprintf("raft serve error:%s",err.Error()))
		return
	}
	log.Info(hostParse.Host)
	ln,err := NewStoppableListener(hostParse.Host, transportStopC)
	if err != nil {
		log.Fatal(fmt.Sprintf("Listener start fails:%s",err.Error()))
	}
	log.Info("Start to serve transport")
	server := &http.Server{Handler:transport.Handler()}
	err = server.Serve(ln)
	select {
	case <-transportStopC:
	default:
		log.Fatal(fmt.Sprintf("Transport server stop : %v",err))

	}
}

func stopTransport(){
	close(transportStopC)
}