package sha3

import (
	"cxchain-2023131076/utils/hash"
	"golang.org/x/crypto/sha3"
)

func Keccak256(value []byte) hash.Hash {
	sha := sha3.NewLegacyKeccak256()
	sha.Write(value)
	return hash.BytesToHash(sha.Sum(nil))
}
