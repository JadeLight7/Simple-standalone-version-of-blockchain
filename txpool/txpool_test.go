package txpool

import (
	"crypto/ecdsa"
	"cxchain-2023131076/crypto"
	"cxchain-2023131076/crypto/secp256k1"
	"cxchain-2023131076/statdb"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/rlp"
	"io"
	"math/big"
	"testing"
	"time"
)

// 问题主要在地址转化和签名
// 内存版 KVStore，仅用于测试
type MemDB struct {
	data map[string][]byte
}

func NewMemDB() *MemDB {
	return &MemDB{data: make(map[string][]byte)}
}
func (m *MemDB) Get(key []byte) ([]byte, error) {
	v, ok := m.data[string(key)]
	if !ok {
		return nil, io.EOF
	}
	return v, nil
}
func (m *MemDB) Put(key, value []byte) error {
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

// 只对 TxData 做 RLP 编码，模拟以太坊的签名前哈希
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

// 用私钥签名并赋值到 Signature 字段
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
		V: v, // 确保类型为byte/uint8
	}
	return nil
}

// 用签名恢复地址
func recoverAddress(tx *types.Transaction) (types.Address, error) {
	hashBytes, err := HashTxData(tx)
	if err != nil {
		return types.Address{}, err
	}
	hash := crypto.Keccak256(hashBytes)
	sig := make([]byte, 65)
	copy(sig[:32], bigIntToBytes32(tx.Signature.R))
	copy(sig[32:64], bigIntToBytes32(tx.Signature.S))
	sig[64] = tx.Signature.V // 确保为byte
	pub, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	addrBytes := crypto.Keccak256(pub[1:])[12:]
	var addr types.Address
	copy(addr[:], addrBytes)
	return addr, nil
}

func TestTxPoolCases(t *testing.T) {
	memdb := NewMemDB()
	db := statdb.NewStatDB(memdb)
	pool := NewDefaultPool(db)

	// 生成私钥和地址
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("密钥生成失败:", err)
	}
	pubKey := privKey.PublicKey
	pubBytes := crypto.FromECDSAPub(&pubKey)
	addr := types.PubKeyToAddress(pubBytes) // 直接传pubBytes（65字节）

	tx1 := &types.Transaction{
		TxData: types.TxData{
			To:       addr, // 先填好
			Nonce:    1,
			Gas:      21000,
			GasPrice: 1,
		},
	}
	if err := signTxWithPriv(tx1, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err := recoverAddress(tx1)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}

	// 校验签名恢复地址
	recovered, err = recoverAddress(tx1)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}

	// 检查 tx1 和 account 的 nonce
	account := db.Load(addr)
	accountNonce := uint64(0)
	if account != nil {
		accountNonce = account.Nonce
	}
	t.Logf("test addr = %x, tx1.From() = %x", addr, tx1.From())
	t.Logf("tx1.Nonce = %d, account.Nonce = %d", tx1.Nonce, accountNonce)
	if tx1.TxData.Nonce != 1 {
		t.Fatalf("tx1 的 nonce 应为 1，实际为 %d", tx1.TxData.Nonce)
	}
	if accountNonce != 0 {
		t.Fatalf("account 的 nonce 应为 0，实际为 %d", accountNonce)
	}

	pool.NewTx(tx1)

	t.Logf("pool.pendings[%x] = %#v", addr, pool.pendings[addr])

	pendingCount := 0
	for _, sortedTxs := range pool.pendings[addr] {
		pendingCount += len(*sortedTxs.(*DefaultSortedTxs))
	}
	if pendingCount == 0 {
		t.Error("tx1 应该进入 pending")
	}

	// 2. nonce 不连续的交易进入 queue
	tx3 := &types.Transaction{
		TxData: types.TxData{
			To:       addr,
			Nonce:    3,
			Gas:      21000,
			GasPrice: 1,
		},
	}
	if err := signTxWithPriv(tx3, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(tx3)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}
	pool.NewTx(tx3)
	if len(pool.queue[addr]) == 0 {
		t.Error("tx3 应该进入 queue，因为 nonce=2 的交易还没到")
	}

	// 3. 补充 nonce=2 的交易，queue 头部交易应自动推进到 pending
	tx2 := &types.Transaction{
		TxData: types.TxData{
			To:       addr,
			Nonce:    2,
			Gas:      21000,
			GasPrice: 1,
		},
	}
	if err := signTxWithPriv(tx2, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(tx2)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}
	pool.NewTx(tx2)
	pendingCount = 0
	for _, sortedTxs := range pool.pendings[addr] {
		pendingCount += len(*sortedTxs.(*DefaultSortedTxs))
	}
	if pendingCount < 3 {
		t.Error("tx2 进入 pending，queue 头部 tx3 nonce 连续，应自动推进到 pending")
	}

	// 4. 测试重复交易（相同 nonce），应替换或拒绝
	tx2dup := &types.Transaction{
		TxData: types.TxData{
			To:       addr,
			Nonce:    2,
			Gas:      21000,
			GasPrice: 2,
		},
	}
	if err := signTxWithPriv(tx2dup, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(tx2dup)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}
	pool.NewTx(tx2dup)
	found := false
	for _, sortedTxs := range pool.pendings[addr] {
		for _, tx := range *sortedTxs.(*DefaultSortedTxs) {
			if tx.Nonce == 2 && tx.GasPrice == 2 {
				found = true
			}
		}
	}
	if !found {
		t.Error("tx2dup 应替换原有 nonce=2 的交易或被拒绝")
	}

	// 5. 测试过期交易清理
	txOld := &types.Transaction{
		TxData: types.TxData{
			To:       addr,
			Nonce:    4,
			Gas:      21000,
			GasPrice: 1,
		},
		Time: uint64(time.Now().Unix()) - 10000,
	}
	if err := signTxWithPriv(txOld, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(txOld)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}
	pool.NewTx(txOld)
	minNonceMap := map[types.Address]uint64{addr: 1}
	pool.CleanExpiredTxs(minNonceMap)
	for _, tx := range pool.queue[addr] {
		if tx.Nonce == 4 {
			t.Error("过期交易未被清理")
		}
	}

	// 6. 测试 pending 满时新交易进入 queue（略）

	// 7. 测试 PopPendingTx 和 PopFromQueue 的正确性
	txPop := pool.PopPendingTx(addr)
	if txPop == nil || txPop.Nonce != 1 {
		t.Error("PopPendingTx 应弹出 pending 队列头部交易 nonce=1")
	} else {
		db.IncNonce(addr) // 关键：同步递增 statdb 里的 nonce
	}
	// 假设 queue 还有交易
	txQ := pool.popFromQueue(&addr)
	if txQ != nil && txQ.Nonce != 4 {
		t.Error("PopFromQueue 应弹出 queue 队列头部交易 nonce=4")
	}

	// 8. 测试多账户并发场景
	privKey2, _ := crypto.GenerateKey()
	pubKey2 := privKey2.PublicKey
	pubBytes2 := crypto.FromECDSAPub(&pubKey2)
	addr2 := types.PubKeyToAddress(pubBytes2[1:]) // 这里同样改为直接传pubBytes2

	txA := &types.Transaction{
		TxData: types.TxData{
			To:       addr2,
			Nonce:    1,
			Gas:      21000,
			GasPrice: 1,
		},
	}
	if err := signTxWithPriv(txA, privKey2); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(txA)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr2 {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr2, recovered)
	}

	txB := &types.Transaction{
		TxData: types.TxData{
			To:       addr,
			Nonce:    1,
			Gas:      21000,
			GasPrice: 1,
		},
	}
	if err := signTxWithPriv(txB, privKey); err != nil {
		t.Fatal("签名失败:", err)
	}
	recovered, err = recoverAddress(txB)
	if err != nil {
		t.Fatal("签名恢复失败:", err)
	}
	if recovered != addr {
		t.Fatalf("签名恢复地址不一致，want %x, got %x", addr, recovered)
	}

	pool.NewTx(txA)
	pool.NewTx(txB)
	pendingA, pendingB := 0, 0
	for _, sortedTxs := range pool.pendings[addr2] {
		pendingA += len(*sortedTxs.(*DefaultSortedTxs))
	}
	for _, sortedTxs := range pool.pendings[addr] {
		pendingB += len(*sortedTxs.(*DefaultSortedTxs))
	}
	if pendingA == 0 || pendingB == 0 {
		t.Error("不同地址的交易应分别维护各自的 pending/queue 队列")
	}
}
