package main

import (
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"log"
	"sync"
	"testing"
)

var db *badger.DB
var local sync.Map

func init() {
	var err error
	opt := badger.DefaultOptions("tmp/badger").WithTableLoadingMode(options.LoadToRAM).WithValueLogLoadingMode(options.FileIO)
	db, err = badger.Open(opt)
	//defer db.Close()
	if err != nil {
		log.Fatal(err)
	}
	local.Store("key1", `{"a":"hello"}`)
}
func Test_Set(t *testing.T) {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("key1"), []byte(`{"a":"hello"}`))
	})
	if err != nil {
		t.Error(err)
	}
}
func Test_Get(t *testing.T) {
	err := db.View(func(txn *badger.Txn) error {
		//txn.NewIterator()
		item, err := txn.Get([]byte("key1"))
		if err != nil {
			t.Error(err)
			return err
		}
		return item.Value(func(val []byte) error {
			t.Logf("key1 ---> %s\n", val)
			return nil
		})
	})
	if err != nil {
		t.Error(err)
	}
}
func Benchmark_Set(b *testing.B) {
	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set([]byte("key1"), []byte(`{"a":"hello"}`))
			})
		}
	})
}
func Benchmark_Get(b *testing.B) {
	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte("key1"))
				if err != nil {
					return err
				}
				return item.Value(func(val []byte) error {
					//fmt.Printf("key1 --- >%s\n",val)
					return nil
				})
			})
		}
	})
}

func Benchmark_GetLocal(b *testing.B) {
	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			local.Load("key1")
		}
	})
}
