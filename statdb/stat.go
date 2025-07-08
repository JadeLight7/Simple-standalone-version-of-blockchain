package statdb

import (
	"cxchain-2023131076/kvstore"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/rlp"
	"fmt"
	"hash"
)

type StatDB interface {
	SetStatRoot(root hash.Hash)                      // 设置状态树根哈希
	Load(addr types.Address) *types.Account          // 加载指定地址的账户信息
	Store(addr types.Address, account types.Account) // 将账户信息存储到状态数据库
	IncNonce(addr types.Address)                     // 递增指定地址账户的 nonce 并保存
}

// DefaultStatDB 是 StatDB 的默认实现，底层使用 kvstore.KVStore
// db: 用于存储状态的底层键值数据库
type DefaultStatDB struct {
	db kvstore.KVStore // 底层存储
}

// NewStatDB 创建一个新的 DefaultStatDB 实例
func NewStatDB(db kvstore.KVStore) *DefaultStatDB {
	return &DefaultStatDB{
		db: db,
	}
}

// SetStatRoot 设置状态树根哈希到数据库
func (s *DefaultStatDB) SetStatRoot(root hash.Hash) {
	bytes, err := rlp.EncodeToBytes(root) // RLP 编码根哈希
	if err != nil {
		fmt.Printf("[DEBUG] RLP 编码失败: %v\n", err)
		return
	}
	s.db.Put([]byte("stat_root"), bytes) // 存储到数据库，key 为 "stat_root"
}

// Load 从数据库加载指定地址的账户信息
func (s *DefaultStatDB) Load(addr types.Address) *types.Account {
	key := addr[:]             // 地址转为字节切片作为 key
	data, err := s.db.Get(key) // 从数据库获取数据
	if err != nil {
		return nil // 获取失败返回 nil
	}
	var account types.Account
	err = rlp.DecodeBytes(data, &account) // RLP 解码账户数据
	if err != nil {
		return nil // 解码失败返回 nil
	}
	return &account // 返回账户指针
}

// Store 将账户信息编码后存储到数据库
func (s *DefaultStatDB) Store(addr types.Address, account types.Account) {
	key := addr[:]
	data, err := rlp.EncodeToBytes(&account)
	if err != nil {
		fmt.Printf("[DEBUG] RLP 编码失败: %v\n", err)
		return
	}
	err = s.db.Put(key, data)
	if err != nil {
		fmt.Printf("[DEBUG] kvstore.Put 失败: %v\n", err)
	}
}

// IncNonce 递增指定地址账户的 nonce 并保存
func (s *DefaultStatDB) IncNonce(addr types.Address) {
	account := s.Load(addr)
	if account == nil {
		account = &types.Account{}
	}
	account.Nonce++
	s.Store(addr, *account)
}
