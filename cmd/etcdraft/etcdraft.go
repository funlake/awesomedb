package etcdraft

import (
	"github.com/funlake/awesomedb/pkg/domain/raftkv"
	"github.com/funlake/awesomedb/pkg/domain/raftkv/rest"
	"github.com/funlake/awesomedb/pkg/log"
	"github.com/spf13/cobra"
)

var (
	peers string
	id    int
	join  bool
	port  string
)

func init() {
	var cp = Command.PersistentFlags()
	cp.StringVar(&peers, "peers", "http://127.0.0.1:12379", "raftkv node peers")
	cp.IntVarP(&id, "id", "i", 1, "Id of raft node")
	cp.BoolVarP(&join, "join", "j", false, "If join to a cluster")
	cp.StringVarP(&port, "port", "p", "12380", "Http serve port")
	if cobra.MarkFlagRequired(cp, "peers") != nil ||
		cobra.MarkFlagRequired(cp, "id") != nil ||
		cobra.MarkFlagRequired(cp, "port") != nil {
		//cobra.MarkFlagRequired(sp, "k") != nil ||
		//cobra.MarkFlagRequired(sp, "ca") != nil {
		log.Error("Fail to set required")
	}
}

var Command = &cobra.Command{
	Use:   "raft.er",
	Short: "start etcd raft",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. etcd-raft,access from http
		raftkv.StartRaftNode(id, join, peers)
		rest.SetUpHttpServer(":" + port)
		log.Info("fuck")
	},
}
