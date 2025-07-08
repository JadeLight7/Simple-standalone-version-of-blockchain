package mpt

import (
	"encoding/hex" // 导入十六进制编码/解码包
)

// toHex 将字节切片转换为十六进制字符串
// 参数 b 是要转换的字节切片
// 返回值是对应的十六进制字符串
func toHex(b []byte) string {
	return hex.EncodeToString(b) // 使用 hex 包的 EncodeToString 函数进行转换
}

// fromHex 将十六进制字符串转换为字节切片
// 参数 s 是要转换的十六进制字符串
// 返回值是对应的字节切片
func fromHex(s string) []byte {
	b, _ := hex.DecodeString(s) // 使用 hex 包的 DecodeString 函数进行转换，忽略错误（假设输入总是有效）
	return b
}

// hexPrefix 对键进行编码处理，添加前缀以区分叶子节点和扩展节点，并处理奇偶性
// 参数 key 是要处理的键的字节切片
// 参数 isLeaf 表示该键是否属于叶子节点
// 返回值是编码后的键的字节切片
func hexPrefix(key []byte, isLeaf bool) []byte {
	var prefixed []byte
	if isLeaf {
		prefixed = append([]byte{0x20}, key...) // 叶子节点键前缀为 0x20
	} else {
		prefixed = append([]byte{0x00}, key...) // 扩展节点键前缀为 0x00
	}
	
	if len(key)%2 == 1 {
		prefixed[0] += 0x10 // 如果键的长度为奇数，设置标志位（将前缀的最高位设置为 1）
	}
	return prefixed
}