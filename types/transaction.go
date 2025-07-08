package types

import (
	"cxchain-2023131076/crypto/secp256k1"
	"cxchain-2023131076/crypto/sha3"
	"cxchain-2023131076/utils/rlp"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

type Receiption struct {
	TxHash hash.Hash
	Status int
	GasUsed uint64
	// Logs
}
type Transaction struct {
	TxData
	Signature Signature
	Time      uint64
}
type TxData struct {
	To       Address
	Nonce    uint64
	Value    uint64
	Gas      uint64
	GasPrice uint64
	Input    []byte
}
type Signature struct {
	R, S *big.Int
	V    uint8
}

func (tx *Transaction) Hash() []byte {
	txdata := tx.TxData
	toSign, err := rlp.EncodeToBytes(txdata)
	if err != nil {
		fmt.Println("error", err)
		return nil
	}
	msg := sha3.Keccak256(toSign)
	return msg[:]
}

func (tx *Transaction) RecoverAddress() (Address, error) {
	msg := tx.Hash() // 32 bytes

	sig := tx.Signature
	signature := make([]byte, 65)
	rBytes := sig.R.Bytes()
	sBytes := sig.S.Bytes()
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)
	recId := sig.V
	if recId >= 27 {
		recId -= 27
	}
	var ErrBadV = errors.New("invalid V value in signature")
	if recId > 1 {
		return Address{}, ErrBadV
	}
	signature[64] = recId

	// 3. 调用底层恢复
	pub, err := secp256k1.RecoverPubkey(msg, signature)
	if err != nil {
		return Address{}, err
	}
	return PubKeyToAddress(pub), nil
}

func (tx *Transaction) From() Address {
	address, err := tx.RecoverAddress()
	if err != nil {
		fmt.Printf(
			"From() failed: v=%d, r=%s, s=%s, nonce=%d, to=%x, hash=%x, err=%v\n",
			tx.Signature.V,
			tx.Signature.R.Text(16),
			tx.Signature.S.Text(16),
			tx.Nonce,
			tx.To[:],
			tx.Hash(),
			err,
		)
		return Address{}
	}
	return address
}
