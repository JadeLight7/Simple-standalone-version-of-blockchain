package leveldb

import (
	"os"
	"testing"
)

func TestLevelDBExample(t *testing.T) {
	// 创建临时目录用于测试数据库
	dbPath := "testdb_example"
	defer os.RemoveAll(dbPath) // 测试完删除数据库目录

	// 创建数据库实例
	store, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	key := []byte("name")
	value := []byte("leveldb")

	// 测试 Put 方法
	if err := store.Put(key, value); err != nil {
		t.Errorf("Put failed: %v", err)
	}

	// 测试 Get 方法
	got, err := store.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if string(got) != "leveldb" {
		t.Errorf("Expected 'leveldb', got %s", got)
	}

	// 测试 Has 方法
	exist, _ := store.Has(key)
	if !exist {
		t.Errorf("Key should exist but not found")
	}

	// 测试 Delete 方法
	if err := store.Delete(key); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// 再次检测 Has
	exist, _ = store.Has(key)
	if exist {
		t.Errorf("Key should have been deleted")
	}
}
