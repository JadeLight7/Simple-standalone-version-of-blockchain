package blockchain

import (
	"cxchain-2023131076/crypto/sha3"
	"cxchain-2023131076/mpt"
	"cxchain-2023131076/txpool"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/hash"
	"cxchain-2023131076/utils/rlp"
)

type Header struct {
	Root       hash.Hash
	ParentHash hash.Hash
	Height     uint64
	Coinbase   types.Address
	Timestamp  uint64
	Difficulty uint64 // 新增：区块难度
	Nonce      uint64
}

type Body struct {
	Transactions []types.Transaction
	Receiptions  []types.Receiption
}

func (header Header) Hash() hash.Hash {
	data, _ := rlp.EncodeToBytes(header)
	return sha3.Keccak256(data)
}

func NewHeader(parent Header) *Header {
	return &Header{
		Root:       parent.Root,
		ParentHash: parent.Hash(),
		Height:     parent.Height + 1,
		Difficulty: parent.Difficulty, // 继承父区块难度
	}
}

func NewBlock() *Body {
	return &Body{
		Transactions: make([]types.Transaction, 0),
		Receiptions:  make([]types.Receiption, 0),
	}
}

type Blockchain struct {
	CurrentHeader Header
	Statedb       mpt.ITrie
	Txpool        txpool.TxPool
}
