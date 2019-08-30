package server

import (
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/funlake/awesomedb/pkg/domain/raftkv"
	"github.com/funlake/awesomedb/pkg/log"

	"github.com/valyala/fasthttp"
)

func SetUpHttpServer(port string) {
	router := fasthttprouter.New()
	router.PUT("/:key", func(ctx *fasthttp.RequestCtx) {
		//log.Info(ctx.UserValue("key").(string))
		//log.Info(string(ctx.PostBody()))
		raftkv.MemoryStorage.Propose(ctx.UserValue("key").(string), string(ctx.PostBody()))
		_, _ = ctx.Write([]byte("ok"))
	})
	router.GET("/:key", func(ctx *fasthttp.RequestCtx) {
		val, _ := raftkv.MemoryStorage.Find(ctx.UserValue("key").(string))
		_, _ = ctx.Write([]byte(val))
	})
	log.Info(fmt.Sprintf("Raft http server listening on port %s", port))
	log.Error(fasthttp.ListenAndServe(port, router.Handler).Error())
}
