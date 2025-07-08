package mpt

import (
	"encoding/json"
	"fmt"
	"crypto/sha256" // 导入 SHA-256 哈希算法包
)

// NodeType 定义节点类型
type NodeType int

// 定义节点类型常量
const (
	Branch NodeType = iota // 分支节点
	Extension          // 扩展节点
	Leaf               // 叶子节点
)

// Node 定义节点接口
type Node interface {
	Type() NodeType // 获取节点类型
	Hash() []byte   // 计算节点哈希
	Serial() ([]byte, error) // 序列化节点
}

// BranchNode 定义分支节点结构
type BranchNode struct {
	Children [17]Node // 17 个子节点（16 个分支 + 1 个值）
}

// 实现 Node 接口的方法
func (n *BranchNode) Type() NodeType { return Branch }
func (n *BranchNode) Hash() []byte  { return hashSerialized(n) }

// ExtensionNode 定义扩展节点结构
type ExtensionNode struct {
	Shared []byte // 共享前缀
	Next   Node   // 下一个节点
}

// 实现 Node 接口的方法
func (n *ExtensionNode) Type() NodeType { return Extension }
func (n *ExtensionNode) Hash() []byte  { return hashSerialized(n) }

// LeafNode 定义叶子节点结构
type LeafNode struct {
	Key   []byte // 键
	Value []byte // 值
}

// 实现 Node 接口的方法
func (n *LeafNode) Type() NodeType { return Leaf }
func (n *LeafNode) Hash() []byte  { return hashSerialized(n) }

// Serial 方法实现序列化分支节点
func (n *BranchNode) Serial() ([]byte, error) {
	return json.Marshal(n)
}

// Serial 方法实现序列化扩展节点
func (n *ExtensionNode) Serial() ([]byte, error) {
	return json.Marshal(n)
}

// Serial 方法实现序列化叶子节点
func (n *LeafNode) Serial() ([]byte, error) {
	return json.Marshal(n)
}

// DeserializeNode 反序列化节点
func DeserializeNode(data []byte) (Node, error) {
	var nodeMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &nodeMap); err != nil {
		return nil, err
	}

	// 根据节点类型反序列化
	if _, ok := nodeMap["Children"]; ok {
		var n BranchNode
		if err := json.Unmarshal(data, &n); err != nil {
			return nil, err
		}
		return &n, nil
	}

	if _, ok := nodeMap["Shared"]; ok {
		var n ExtensionNode
		if err := json.Unmarshal(data, &n); err != nil {
			return nil, err
		}
		return &n, nil
	}

	if _, ok := nodeMap["Key"]; ok {
		var n LeafNode
		if err := json.Unmarshal(data, &n); err != nil {
			return nil, err
		}
		return &n, nil
	}

	return nil, fmt.Errorf("unknown node type")
}

// hashSerialized 计算序列化后的节点哈希
func hashSerialized(n Node) []byte {
	data, _ := n.Serial()
	return hash(data)
}

// hash 计算哈希值
func hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// equal 比较两个字节数组是否相等
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// min 返回两个整数中的较小者
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// keyToHex 将键转换为十六进制表示
func keyToHex(key []byte) []byte {
	hexKey := make([]byte, len(key)*2)
	for i, b := range key {
		hexKey[i*2] = b >> 4
		hexKey[i*2+1] = b & 0x0F
	}
	return hexKey
}