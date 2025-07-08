package mpt

import (
	"cxchain-2023131076/kvstore"
)

// TrieDB 定义了 MPT 数据库的结构，用于存储和加载节点
type TrieDB struct {
	store kvstore.KVStore // 底层键值存储
}

// NewTrieDB 创建一个新的 MPT 数据库实例
// 参数 store 是底层的键值存储实现
// 返回值是一个指向 TrieDB 结构的指针
func NewTrieDB(store kvstore.KVStore) *TrieDB {
	return &TrieDB{store: store} // 初始化并返回 TrieDB 实例
}

// StoreNode 将节点存储到数据库中
// 参数 node 是要存储的节点
// 返回值是一个错误，如果存储过程中出现问题则不为 nil
func (db *TrieDB) StoreNode(node Node) error {
	hash := node.Hash() // 计算节点的哈希值
	data, err := node.Serial() // 序列化节点数据
	if err != nil {
		return err // 如果序列化失败，返回错误
	}
	return db.store.Put(hash, data) // 将节点数据存储到数据库中，键为哈希值
}

// LoadNode 从数据库中加载节点
// 参数 hash 是节点的哈希值
// 返回值是一个 Node 接口类型的节点和一个错误
func (db *TrieDB) LoadNode(hash []byte) (Node, error) {
	data, err := db.store.Get(hash) // 从数据库中获取节点数据
	if err != nil {
		return nil, err // 如果获取失败，返回错误
	}
	return DeserializeNode(data) // 反序列化节点数据并返回
}