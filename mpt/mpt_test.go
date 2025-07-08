package mpt

import (
	"testing"
	. "cxchain-2023131076/kvstore"
)

// TestMPTBasic 测试 MPT 树的基本功能
func TestMPTBasic(t *testing.T) {
	db, err := NewLevelDB("testdb") // 初始化 LevelDB 数据库
	if err != nil {
		t.Fatalf("Failed to open LevelDB: %v", err) // 如果初始化失败，终止测试并输出错误信息
	}
	defer db.Close() // 测试结束后关闭数据库

	trie := NewTrie(db) // 创建一个新的 MPT 树实例

	key := []byte("testKey")   // 定义测试用的键
	value := []byte("testValue") // 定义测试用的值
	trie.Insert(key, value) // 将键值对插入 MPT 树

	retrieved, err := trie.Get(key) // 尝试从 MPT 树中获取值
	if err != nil {
		t.Fatalf("Get failed: %v", err) // 如果获取失败，终止测试并输出错误信息
	}

	if string(retrieved) != string(value) { // 比较获取到的值和预期值
		t.Errorf("Expected %s, got %s", value, retrieved) // 如果不匹配，输出错误信息
	}

	trie.Delete(key) // 从 MPT 树中删除键值对
	_, err = trie.Get(key) // 尝试获取已删除的键值对
	if err == nil { // 如果没有错误发生
		t.Error("Expected error after deletion") // 输出错误信息
	}

	trie.Insert([]byte("key1"), []byte("value1")) // 插入第一个键值对
	hash1 := trie.RootHash() // 获取根哈希

	trie.Insert([]byte("key2"), []byte("value2")) // 插入第二个键值对
	hash2 := trie.RootHash() // 获取根哈希

	if equal(hash1, hash2) { // 比较两次插入后的根哈希
		t.Error("Root hashes should be different after modification") // 如果相同，输出错误信息
	}
}