package mpt

import (
	"errors"
	"cxchain-2023131076/kvstore"
)

// Trie 定义了 MPT 树的结构
type Trie struct {
	root Node // 根节点
	db   *TrieDB // 数据库存储
}
type ITrie interface {
    Get(key []byte) ([]byte, error)
    Insert(key, value []byte) error
    Delete(key []byte) error
    RootHash() []byte
}
// NewTrie 创建一个新的 MPT 树实例
func NewTrie(db kvstore.KVStore) *Trie {
	return &Trie{
		db: NewTrieDB(db),
	}
}

// Get 根据键获取值
func (t *Trie) Get(key []byte) ([]byte, error) {
	hexKey := keyToHex(key) // 将键转换为十六进制表示
	current := t.root

	for current != nil {
		switch n := current.(type) {
		case *BranchNode: // 如果当前节点是分支节点
			if len(hexKey) == 0 { // 如果键已经匹配完成
				if n.Children[16] != nil { // 如果存在值节点
					if leaf, ok := n.Children[16].(*LeafNode); ok { // 尝试将其转换为叶子节点
						return leaf.Value, nil
					}
				}
				return nil, errors.New("key not found")
			}
			idx := hexKey[0] // 获取当前键的第一个字节
			if int(idx) >= len(n.Children) || n.Children[idx] == nil { // 如果没有对应的子节点
				return nil, errors.New("key not found")
			}
			current = n.Children[idx] // 移动到子节点
			hexKey = hexKey[1:] // 剩余的部分继续匹配

		case *ExtensionNode: // 如果当前节点是扩展节点
			shared := n.Shared // 获取共享前缀
			if len(hexKey) < len(shared) || !equal(hexKey[:len(shared)], shared) { // 如果键与共享前缀不匹配
				return nil, errors.New("key not found")
			}
			current = n.Next // 移动到下一个节点
			hexKey = hexKey[len(shared):] // 跳过共享前缀部分

		case *LeafNode: // 如果当前节点是叶子节点
			if equal(hexKey, n.Key) { // 如果键完全匹配
				return n.Value, nil
			}
			return nil, errors.New("key not found")
		}
	}

	return nil, errors.New("key not found")
}

// Insert 插入键值对
func (t *Trie) Insert(key, value []byte) error {
	hexKey := keyToHex(key) // 将键转换为十六进制表示
	var err error
	t.root, err = t.insert(t.root, hexKey, value) // 插入操作
	if err != nil {
		return err
	}
	return t.db.StoreNode(t.root) // 将节点存储到数据库中
}

// insert 辅助函数，用于递归插入节点
func (t *Trie) insert(node Node, key []byte, value []byte) (Node, error) {
	if node == nil { // 如果当前节点为空，创建一个新的叶子节点
		return &LeafNode{Key: key, Value: value}, nil
	}

	switch n := node.(type) {
	case *BranchNode: // 当前节点是分支节点
		if len(key) == 0 { // 如果键已经匹配完成
			n.Children[16] = &LeafNode{Key: nil, Value: value} // 存储值在最后一个子节点位置
			return n, nil
		}

		idx := key[0] // 获取当前键的第一个字节
		child, err := t.insert(n.Children[idx], key[1:], value) // 递归插入到子节点
		if err != nil {
			return nil, err
		}
		n.Children[idx] = child // 更新子节点
		return n, nil

	case *ExtensionNode: // 当前节点是扩展节点
		sharedPrefix := 0
		minLen := min(len(key), len(n.Shared)) // 计算键和共享前缀的最小长度
		for ; sharedPrefix < minLen; sharedPrefix++ { // 寻找共享前缀的匹配长度
			if key[sharedPrefix] != n.Shared[sharedPrefix] {
				break
			}
		}

		if sharedPrefix == len(n.Shared) { // 如果共享前缀完全匹配
			next, err := t.insert(n.Next, key[sharedPrefix:], value) // 递归插入到下一个节点
			if err != nil {
				return nil, err
			}
			n.Next = next // 更新下一个节点
			return n, nil
		}

		branch := &BranchNode{} // 创建一个新的分支节点

		splitIdx := n.Shared[sharedPrefix] // 获取分叉点的索引

		if sharedPrefix+1 < len(n.Shared) { // 如果还有剩余的共享前缀
			branch.Children[splitIdx] = &ExtensionNode{ // 创建新的扩展节点
				Shared: n.Shared[sharedPrefix+1:],
				Next:   n.Next,
			}
		} else {
			branch.Children[splitIdx] = n.Next // 直接连接到下一个节点
		}

		if sharedPrefix < len(key) { // 如果键还有剩余部分
			newKeyIdx := key[sharedPrefix] // 获取新的键索引
			if sharedPrefix+1 < len(key) { // 如果键还有更多部分
				branch.Children[newKeyIdx] = &LeafNode{ // 创建新的叶子节点
					Key:   key[sharedPrefix+1:],
					Value: value,
				}
			} else {
				branch.Children[16] = &LeafNode{Value: value} // 将值存储在最后一个子节点位置
			}
		} else {
			branch.Children[16] = &LeafNode{Value: value} // 将值存储在最后一个子节点位置
		}

		if sharedPrefix > 0 { // 如果有共享前缀，创建新的扩展节点
			return &ExtensionNode{
				Shared: n.Shared[:sharedPrefix],
				Next:   branch,
			}, nil
		}

		return branch, nil

	case *LeafNode: // 当前节点是叶子节点
		sharedPrefix := 0
		minLen := min(len(key), len(n.Key)) // 计算键和叶子节点键的最小长度
		for ; sharedPrefix < minLen; sharedPrefix++ { // 寻找共享前缀的匹配长度
			if key[sharedPrefix] != n.Key[sharedPrefix] {
				break
			}
		}

		if sharedPrefix == len(key) && sharedPrefix == len(n.Key) { // 如果键完全匹配
			n.Value = value // 更新值
			return n, nil
		}

		branch := &BranchNode{} // 创建一个新的分支节点

		if sharedPrefix < len(n.Key) { // 如果叶子节点键还有剩余部分
			leafIdx := n.Key[sharedPrefix] // 获取叶子节点的键索引
			if sharedPrefix+1 < len(n.Key) { // 如果叶子节点键还有更多部分
				branch.Children[leafIdx] = &LeafNode{ // 创建新的叶子节点
					Key:   n.Key[sharedPrefix+1:],
					Value: n.Value,
				}
			} else {
				branch.Children[16] = &LeafNode{Value: n.Value} // 将值存储在最后一个子节点位置
			}
		} else {
			branch.Children[16] = n // 直接连接到叶子节点
		}

		if sharedPrefix < len(key) { // 如果键还有剩余部分
			newKeyIdx := key[sharedPrefix] // 获取新的键索引
			if sharedPrefix+1 < len(key) { // 如果键还有更多部分
				branch.Children[newKeyIdx] = &LeafNode{ // 创建新的叶子节点
					Key:   key[sharedPrefix+1:],
					Value: value,
				}
			} else {
				branch.Children[16] = &LeafNode{Value: value} // 将值存储在最后一个子节点位置
			}
		} else {
			branch.Children[16] = &LeafNode{Value: value} // 将值存储在最后一个子节点位置
		}

		if sharedPrefix > 0 { // 如果有共享前缀，创建新的扩展节点
			return &ExtensionNode{
				Shared: key[:sharedPrefix],
				Next:   branch,
			}, nil
		}

		return branch, nil
	}

	return nil, errors.New("unknown node type")
}

// Delete 删除键值对
func (t *Trie) Delete(key []byte) error {
	hexKey := keyToHex(key) // 将键转换为十六进制表示
	var deleted bool
	var err error
	t.root, deleted, err = t.delete(t.root, hexKey) // 删除操作
	if err != nil {
		return err
	}
	if !deleted {
		return errors.New("key not found")
	}
	return nil
}

// delete 辅助函数，用于递归删除节点
func (t *Trie) delete(node Node, key []byte) (Node, bool, error) {
	if node == nil { // 如果当前节点为空，返回未删除
		return nil, false, nil
	}

	switch n := node.(type) {
	case *LeafNode: // 当前节点是叶子节点
		if equal(n.Key, key) { // 如果键匹配，删除该叶子节点
			return nil, true, nil
		}
		return n, false, nil

	case *ExtensionNode: // 当前节点是扩展节点
		shared := n.Shared // 获取共享前缀
		if len(key) < len(shared) || !equal(key[:len(shared)], shared) { // 如果键与共享前缀不匹配
			return n, false, nil
		}
		next, deleted, err := t.delete(n.Next, key[len(shared):]) // 递归删除下一个节点
		if err != nil {
			return nil, false, err
		}
		if !deleted {
			return n, false, nil
		}
		// 如果下一个节点为空，删除扩展节点
		if next == nil {
			return nil, true, nil
		}
		// 如果下一个节点是叶子节点或扩展节点，合并路径
		switch nextNode := next.(type) {
		case *ExtensionNode:
			return &ExtensionNode{
				Shared: append(shared, nextNode.Shared...),
				Next:   nextNode.Next,
			}, true, nil
		case *LeafNode:
			return &ExtensionNode{
				Shared: append(shared, nextNode.Key...),
				Next:   nextNode,
			}, true, nil
		default:
			return &ExtensionNode{
				Shared: shared,
				Next:   next,
			}, true, nil
		}

	case *BranchNode: // 当前节点是分支节点
		if len(key) == 0 { // 如果键已经匹配完成
			n.Children[16] = nil // 删除值节点
		} else {
			idx := key[0] // 获取当前键的第一个字节
			child, deleted, err := t.delete(n.Children[idx], key[1:]) // 递归删除子节点
			if err != nil {
				return nil, false, err
			}
			if !deleted {
				return n, false, nil
			}
			n.Children[idx] = child // 更新子节点
		}

		// 统计非空子节点数
		var count int
		var lastIdx int
		for i, child := range n.Children {
			if child != nil {
				count++
				lastIdx = i
			}
		}

		// 如果没有子节点，删除整个分支
		if count == 0 {
			return nil, true, nil
		}

		// 如果只有一个子节点，转换为叶子节点或扩展节点
		if count == 1 {
			child := n.Children[lastIdx]
			if lastIdx == 16 { // 如果是值节点
				if leaf, ok := child.(*LeafNode); ok {
					return &LeafNode{
						Key:   []byte{},
						Value: leaf.Value,
					}, true, nil
				}
				return child, true, nil
			}
			switch c := child.(type) {
			case *LeafNode:
				return &LeafNode{
					Key:   append([]byte{byte(lastIdx)}, c.Key...),
					Value: c.Value,
				}, true, nil
			case *ExtensionNode:
				return &ExtensionNode{
					Shared: append([]byte{byte(lastIdx)}, c.Shared...),
					Next:   c.Next,
				}, true, nil
			default:
				return &ExtensionNode{
					Shared: []byte{byte(lastIdx)},
					Next:   c,
				}, true, nil
			}
		}

		return n, true, nil
	}

	return node, false, nil
}

// RootHash 获取 MPT 树的根哈希
func (t *Trie) RootHash() []byte {
	if t.root == nil { // 如果根节点为空，返回空哈希
		return nil
	}
	return t.root.Hash() // 返回根节点的哈希
}