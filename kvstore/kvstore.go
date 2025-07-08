package kvstore

import (
	"io"
	"github.com/syndtr/goleveldb/leveldb" // 导入 LevelDB 包
)

// KVStore 定义了一个通用的键值存储接口
type KVStore interface {
	Get(key []byte) ([]byte, error)   // 获取指定键的值
	Put(key, value []byte) error      // 存储键值对
	Delete(key []byte) error          // 删除指定键
	Has(key []byte) (bool, error)     // 检查键是否存在
	io.Closer                         // 定义了 Close 方法，用于释放资源
}

// LevelDB 定义了一个基于 LevelDB 的键值存储实现
type LevelDB struct {
	db *leveldb.DB // 底层的 LevelDB 数据库实例
}

// NewLevelDB 创建一个新的 LevelDB 实例
// 参数 path 是数据库文件的存储路径
// 返回值是一个 LevelDB 实例和错误（如果创建失败则不为 nil）
func NewLevelDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil) // 打开或创建 LevelDB 数据库文件
	if err != nil {
		return nil, err // 如果打开失败，返回错误
	}
	return &LevelDB{db: db}, nil // 返回 LevelDB 实例
}

// Get 实现了 KVStore 接口的 Get 方法，用于获取指定键的值
// 参数 key 是要查询的键
// 返回值是键对应的值和错误（如果查询失败则不为 nil）
func (ldb *LevelDB) Get(key []byte) ([]byte, error) {
	return ldb.db.Get(key, nil) // 使用 LevelDB 的 Get 方法查询键值
}

// Put 实现了 KVStore 接口的 Put 方法，用于存储键值对
// 参数 key 是键，value 是值
// 返回值是错误（如果存储失败则不为 nil）
func (ldb *LevelDB) Put(key, value []byte) error {
	return ldb.db.Put(key, value, nil) // 使用 LevelDB 的 Put 方法存储键值对
}

// Delete 实现了 KVStore 接口的 Delete 方法，用于删除指定键
// 参数 key 是要删除的键
// 返回值是错误（如果删除失败则不为 nil）
func (ldb *LevelDB) Delete(key []byte) error {
	return ldb.db.Delete(key, nil) // 使用 LevelDB 的 Delete 方法删除键
}

// Has 实现了 KVStore 接口的 Has 方法，用于检查键是否存在
// 参数 key 是要检查的键
// 返回值是一个布尔值（存在为 true）和错误（如果检查失败则不为 nil）
func (ldb *LevelDB) Has(key []byte) (bool, error) {
	return ldb.db.Has(key, nil) // 使用 LevelDB 的 Has 方法检查键是否存在
}

// Close 实现了 io.Closer 接口的 Close 方法，用于关闭数据库并释放资源
// 返回值是错误（如果关闭失败则不为 nil）
func (ldb *LevelDB) Close() error {
	return ldb.db.Close() // 使用 LevelDB 的 Close 方法关闭数据库
}