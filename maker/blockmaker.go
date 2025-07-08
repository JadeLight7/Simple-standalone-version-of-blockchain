package maker

import (
	"cxchain-2023131076/blockchain"
	"cxchain-2023131076/statdb"
	"cxchain-2023131076/statemachine"
	"cxchain-2023131076/txpool"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/xtime"
	"fmt"
	"time"
)

// ChainConfig 区块链链参数配置
type ChainConfig struct {
	Duration   time.Duration
	Coinbase   types.Address
	Difficulty uint64
}

// BlockMaker 区块生产器，负责从交易池打包区块
// txpool: 交易池接口
// state: 状态数据库
// exec: 状态机执行器
// config: 链参数
// chain: 区块链对象
// nextHeader/nextBody: 当前正在构建的区块头和区块体
// interupt: 用于打包协程的中断信号
type BlockMaker struct {
	txpool txpool.TxPool
	state  statdb.StatDB
	exec   statemachine.IMachine

	config ChainConfig
	chain  blockchain.Blockchain

	nextHeader *blockchain.Header
	nextBody   *blockchain.Body

	interupt chan bool // 通道，用于不同携程之间的
}

// NewBlockMaker 创建 BlockMaker 实例
func NewBlockMaker(txpool txpool.TxPool, state statdb.StatDB, exec statemachine.IMachine, config ChainConfig, chain blockchain.Blockchain) *BlockMaker {
	return &BlockMaker{
		txpool:     txpool,
		state:      state,
		exec:       exec,
		config:     config,
		chain:      chain,
		nextHeader: nil,
		nextBody:   nil,
		interupt:   make(chan bool, 1),
	}
}

// NewBlock 初始化新区块头和区块体
func (maker *BlockMaker) NewBlock() {
	maker.nextBody = blockchain.NewBlock()
	maker.nextHeader = blockchain.NewHeader(maker.chain.CurrentHeader)
	maker.nextHeader.Coinbase = maker.config.Coinbase
	// 动态设置难度（假设区块头有 Difficulty 字段）
	maker.nextHeader.Difficulty = maker.config.Difficulty
	maker.nextHeader.Timestamp = xtime.Now()
	// 其它字段可根据需要初始化
}

// Pack 启动区块打包主循环，直到超时或被中断
func (maker *BlockMaker) Pack() {
	for {
		if !maker.pack() {
			break
		}
	}
}

// pack 尝试打包一笔交易，返回是否成功
func (maker *BlockMaker) pack() bool {
	tx := maker.txpool.Pop()
	if tx == nil {
		fmt.Println("[DEBUG] Pop 返回 nil，pending 为空或无可打包交易")
		return false
	}
	fmt.Printf("[DEBUG] 打包交易 nonce=%d, from=%x, to=%x, value=%d\n", tx.Nonce, tx.From(), tx.To, tx.Value)
	receiption := maker.exec.Execute(maker.state, *tx)
	if receiption == nil || receiption.Status == 0 {
		fmt.Printf("[DEBUG] 交易执行失败 nonce=%d, status=%v\n", tx.Nonce, receiption)
		return false
	}
	maker.nextBody.Transactions = append(maker.nextBody.Transactions, *tx)
	maker.nextBody.Receiptions = append(maker.nextBody.Receiptions, *receiption)
	return true
}

// Interupt 发送中断信号，安全退出打包循环
func (maker *BlockMaker) Interupt() {
	select {
	case maker.interupt <- true:
	default:
	}
}

// Finalize 完成区块，返回区块头和区块体，并重置内部状态
func (maker *BlockMaker) Finalize() (*blockchain.Header, *blockchain.Body) {
	maker.nextHeader.Timestamp = xtime.Now()
	// ...难度和nonce逻辑略...
	head := maker.nextHeader
	body := maker.nextBody
	maker.nextHeader = nil
	maker.nextBody = nil
	return head, body
}
