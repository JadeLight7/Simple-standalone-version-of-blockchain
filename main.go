package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"cxchain-2023131076/blockchain"
	"cxchain-2023131076/crypto"
	"cxchain-2023131076/crypto/secp256k1"
	"cxchain-2023131076/maker"
	"cxchain-2023131076/statdb"
	"cxchain-2023131076/statemachine"
	"cxchain-2023131076/txpool"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/rlp"
)

// 内存版 KVStore，仅用于测试
type MemDB struct {
	data map[string][]byte
}

func NewMemDB() *MemDB {
	return &MemDB{data: make(map[string][]byte)}
}
func (m *MemDB) Get(key []byte) ([]byte, error) {
	fmt.Printf("[DEBUG] MemDB.Get: key=%x\n", key)
	v, ok := m.data[string(key)]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return v, nil
}
func (m *MemDB) Put(key, value []byte) error {
	fmt.Printf("[DEBUG] MemDB.Put: key=%x, value=%x\n", key, value)
	m.data[string(key)] = value
	return nil
}
func (m *MemDB) Delete(key []byte) error {
	delete(m.data, string(key))
	return nil
}
func (m *MemDB) Has(key []byte) (bool, error) {
	_, ok := m.data[string(key)]
	return ok, nil
}
func (m *MemDB) Close() error { return nil }

// 只对 TxData 做 RLP 编码
func HashTxData(tx *types.Transaction) ([]byte, error) {
	return rlp.EncodeToBytes(tx.TxData)
}

func bigIntToBytes32(b *big.Int) []byte {
	bs := b.Bytes()
	if len(bs) > 32 {
		return bs[len(bs)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(bs):], bs)
	return padded
}

// 用私钥签名并赋值到 Signature 字段（参考交易池测试）
func signTxWithPriv(tx *types.Transaction, priv *ecdsa.PrivateKey) error {
	hashBytes, err := HashTxData(tx)
	if err != nil {
		return err
	}
	hash := crypto.Keccak256(hashBytes)
	seckey := crypto.FromECDSA(priv)
	sig, err := secp256k1.Sign(hash, seckey)
	if err != nil {
		return err
	}
	v := sig[64]
	if v >= 27 {
		v = v - 27
	}
	tx.Signature = types.Signature{
		R: new(big.Int).SetBytes(sig[:32]),
		S: new(big.Int).SetBytes(sig[32:64]),
		V: v,
	}
	return nil
}

// 用签名恢复地址（参考交易池测试）
func recoverAddress(tx *types.Transaction) (types.Address, error) {
	hashBytes, err := HashTxData(tx)
	if err != nil {
		return types.Address{}, err
	}
	hash := crypto.Keccak256(hashBytes)
	sig := make([]byte, 65)
	copy(sig[:32], bigIntToBytes32(tx.Signature.R))
	copy(sig[32:64], bigIntToBytes32(tx.Signature.S))
	sig[64] = tx.Signature.V
	pub, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	addrBytes := crypto.Keccak256(pub[1:])[12:]
	var addr types.Address
	copy(addr[:], addrBytes)
	return addr, nil
}

func countPending(pool *txpool.DefaultPool, addr types.Address) int {
	pendingCount := 0
	for _, sortedTxs := range pool.GetPendings()[addr] {
		pendingCount += len(*sortedTxs.(*txpool.DefaultSortedTxs))
	}
	return pendingCount
}

func main() {
	// 1. 初始化
	memdb := NewMemDB()
	db := statdb.NewStatDB(memdb)
	pool := txpool.NewDefaultPool(db)
	exec := &statemachine.StateMachine{}
	chain := &blockchain.Blockchain{}
	config := maker.ChainConfig{
		Duration:   2 * time.Second,
		Coinbase:   types.Address{},
		Difficulty: 8,
	}
	blockMaker := maker.NewBlockMaker(pool, db, exec, config, *chain)

	// 2. 生成账户和交易
	privKey, _ := crypto.GenerateKey()
	pubKey := privKey.PublicKey
	pubBytes := crypto.FromECDSAPub(&pubKey)
	addr := types.PubKeyToAddress(pubBytes)
	fmt.Printf("[INFO] 账户地址: %x\n", addr)

	// 给账户预置余额
	account := &types.Account{
		Nonce:  0,
		Amount: 100000000,
	}
	db.Store(addr, *account)
	acc := db.Load(addr)
	fmt.Printf("[DEBUG] 写后立即读: addr=%x, account=%+v\n", addr, acc)

	// 构造多笔交易
	for i := 1; i <= 3; i++ {
		tx := &types.Transaction{
			TxData: types.TxData{
				To:       addr,
				Nonce:    uint64(i),
				Gas:      21000,
				GasPrice: 1,
				Value:    10,
			},
		}
		signTxWithPriv(tx, privKey)
		pool.NewTx(tx)
		fmt.Printf("[INFO] 交易%d已加入交易池，nonce=%d\n", i, i)
	}

	fmt.Printf("[INFO] 当前pending数: %d\n", countPending(pool, addr))

	// 3. 出块流程
	blockMaker.NewBlock()
	blockMaker.Pack() // 会循环 Pop()，直到pending为空
	head, body := blockMaker.Finalize()
	fmt.Printf("[INFO] 新区块已打包，包含交易数：%d\n", len(body.Transactions))

	// 4. 区块链记录区块
	chain.CurrentHeader = *head // 只简单赋值
	fmt.Printf("[INFO] 区块高度: %d, 状态根: %x\n", head.Height, head.Root)

	// 5. 交易池推进/清理（如无 RemoveTx 可省略）
	// for _, tx := range body.Transactions {
	//     pool.RemoveTx(tx)
	// }
	fmt.Printf("[INFO] 区块上链后pending数: %d\n", countPending(pool, addr))

	// 6. 查询账户余额
	accountAfter := db.Load(addr)
	if accountAfter != nil {
		fmt.Printf("[INFO] 区块链执行后账户余额: %d, nonce: %d\n", accountAfter.Amount, accountAfter.Nonce)
	} else {
		fmt.Println("[WARN] 未找到账户")
	}

	fmt.Println("[INFO] 单机区块链主流程演示完毕。")
}
