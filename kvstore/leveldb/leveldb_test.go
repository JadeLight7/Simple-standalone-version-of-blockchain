package leveldb

import (
	"os"
	"strconv"
	"testing"
)

// BenchmarkPut 用于测试 Put 操作的性能
func BenchmarkPut(b *testing.B) {
	dbPath := "testdb_bench"
	defer os.RemoveAll(dbPath)

	store, _ := NewLevelDB(dbPath)
	defer store.Close()

	for i := 0; i < b.N; i++ {
		key := []byte("key" + strconv.Itoa(i))
		value := []byte("value" + strconv.Itoa(i))
		_ = store.Put(key, value)
	}
}
