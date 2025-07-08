package types

import (
	"cxchain-2023131076/crypto/sha3"
)

type Address [20]byte

func PubKeyToAddress(pub []byte) Address {
	if len(pub) == 65 && pub[0] == 0x04 {
		pub = pub[1:] // 只取X+Y
	}
	if len(pub) < 64 {
		return Address{}
	}
	h := sha3.Keccak256(pub)
	var addr Address
	copy(addr[:], h[12:32])
	return addr
}
