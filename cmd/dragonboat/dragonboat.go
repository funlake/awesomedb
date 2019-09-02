package dragonboat

import (
	"github.com/funlake/awesomedb/pkg/domain/dbraft"
	"github.com/funlake/awesomedb/pkg/log"
	"github.com/spf13/cobra"
)

var (
	peers string
	id    int
	join  bool
)

func init() {
	var cp = Command.PersistentFlags()
	cp.StringVar(&peers, "peers", "127.0.0.1:12379", "raftkv node peers")
	cp.IntVarP(&id, "id", "i", 1, "Id of raft node")
	cp.BoolVarP(&join, "join", "j", false, "If join to a cluster")
	//cp.StringVarP(&port, "port", "p", "12380", "Http serve port")
	if cobra.MarkFlagRequired(cp, "peers") != nil ||
		cobra.MarkFlagRequired(cp, "id") != nil {
		log.Error("Fail to set required")
	}
}

var Command = &cobra.Command{
	Use:   "raft.dr",
	Short: "Setup dragon boat raft",
	Run: func(cmd *cobra.Command, args []string) {
		//1. dragonboat-raft
		//access from console command line
		dbraft.StartMemoryRaftNode(id, join, peers)
	},
}
