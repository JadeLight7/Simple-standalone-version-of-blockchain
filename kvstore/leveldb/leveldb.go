package leveldb

import (
	"cxchain-2023131076/kvstore"              // 导入接口定义
	"github.com/syndtr/goleveldb/leveldb"     // 引入 goleveldb 库
)

// LevelDB 是对底层 leveldb.DB 的封装，实现了 kvstore.KVStore 接口
type LevelDB struct {
	db *leveldb.DB
}

// NewLevelDB 打开或创建一个 LevelDB 数据库实例
func NewLevelDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil) // nil 表示使用默认配置
	if err != nil {
		return nil, err
	}
	return &LevelDB{db: db}, nil
}

// Get 根据 key 获取对应的 value
func (l *LevelDB) Get(key []byte) ([]byte, error) {
	return l.db.Get(key, nil)
}

// Put 存储 key-value 对
func (l *LevelDB) Put(key, value []byte) error {
	return l.db.Put(key, value, nil)
}

// Delete 删除指定 key 的记录
func (l *LevelDB) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

// Has 判断 key 是否存在于数据库中
func (l *LevelDB) Has(key []byte) (bool, error) {
	return l.db.Has(key, nil)
}

// Close 关闭数据库，释放文件句柄等资源
func (l *LevelDB) Close() error {
	return l.db.Close()
}

// 确保 LevelDB 实现了 kvstore.KVStore 接口
var _ kvstore.KVStore = (*LevelDB)(nil)
