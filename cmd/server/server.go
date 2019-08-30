package server

import (
	"github.com/funlake/awesomedb/pkg/domain/raftkv"
	"github.com/funlake/awesomedb/pkg/domain/raftkv/server"
	"github.com/spf13/cobra"
)
var (
	peers string
	id int
	join bool
	port string
)
func init() {
	var cp = Command.PersistentFlags()
	cp.StringVar(&peers,"peers","http://127.0.0.1:12379","raftkv node peers")
	cp.IntVarP(&id,"id","i",1,"Id of raft node")
	cp.BoolVarP(&join,"join","j",false,"If join to a cluster")
	cp.StringVarP(&port,"port","p","8898","Http serve port")


}
var Command = &cobra.Command{
	Use: "start.db",
	Short: "start db node",
	Run : func(cmd *cobra.Command, args []string) {
		raftkv.StartRaftNode(id,join,peers)
		server.SetUpHttpServer(":"+port)
	},
}