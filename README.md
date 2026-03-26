
# 🚀 Simple Standalone Blockchain (Go)

一个使用 Go 实现的**单机版区块链系统**，涵盖区块链的核心模块，包括：

* 区块生成
* 交易处理
* 状态管理（账户模型）
* 交易池（TxPool）
* 状态机执行
---

# 📦 项目结构

```
├── blockchain/     # 区块链核心（区块结构 + 链管理）
├── crypto/         # 加密模块（签名 / 地址生成）
├── kvstore/        # 存储接口抽象（可扩展）
├── maker/          # 区块生成器（打包逻辑）
├── mpt/            # Merkle Patricia Trie（状态树）
├── statdb/         # 状态数据库
├── statemachine/   # 状态机（交易执行）
├── txpool/         # 交易池（pending / queued）
├── types/          # 核心数据结构定义
├── utils/          # 工具函数
├── main.go         # 程序入口
├── go.mod
└── go.sum
```

---

# ⚙️ 核心功能

## 🧱 1. 区块链结构

* 区块头 + 区块体设计
* 基于哈希的链式结构（防篡改）
* 简单 PoW 难度支持

---

## 💸 2. 交易系统

* 交易创建 + ECDSA 签名
* 支持地址恢复（类似 Ethereum）
* TxPool 设计：

  * pending（可执行）
  * queued（等待 nonce）
* 按 `GasPrice` 排序优先打包

---

## 🌳 3. 状态管理

* 账户模型（余额 + nonce）
* 基于 KVStore 抽象存储
* 使用 MPT（Merkle Patricia Trie）组织状态

👉 类似 Ethereum Virtual Machine 状态结构

---

## ⛏️ 4. 区块生成

* 从 TxPool 挑选交易
* 打包区块
* 区块最终化（Finalize）
* 写入链

---

## 🔁 5. 状态机执行

* 交易执行（转账 / nonce）
* Gas 消耗计算
* 状态更新（State Transition）

---

# 🏗️ 系统流程

```
用户创建交易
    ↓
交易进入 TxPool
    ↓
区块生成器打包交易
    ↓
状态机执行交易
    ↓
更新状态（MPT）
    ↓
新区块写入链
```

---

# 🚀 快速开始

## 环境要求

* Go 1.16+

---

## 运行项目

```bash
git clone https://github.com/yourusername/simple-blockchain.git
cd simple-blockchain
go run main.go
```

# 💻 示例代码

## 创建并签名交易

```go
privKey, _ := crypto.GenerateKey()

pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
addr := types.PubKeyToAddress(pubBytes)

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
```

---

## 区块生成流程

```go
blockMaker := maker.NewBlockMaker(pool, db, exec, config, *chain)

blockMaker.NewBlock()
blockMaker.Pack()

head, body := blockMaker.Finalize()

chain.CurrentHeader = *head
```

---


