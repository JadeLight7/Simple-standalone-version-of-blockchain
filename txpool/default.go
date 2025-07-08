package txpool

import (
	"cxchain-2023131076/statdb"
	"cxchain-2023131076/types"
	"cxchain-2023131076/utils/xtime"
	"hash"
	"sort"
	"time"
)

type SortedTxs interface {
	GasPrice() uint64//盒子里的第一个交易gas值，同一个盒子中的gas只需大于等于第一个就行
	Push(tx *types.Transaction)
	Pop() *types.Transaction
	Replace(tx *types.Transaction)
	FirstNonce() uint64
	LastNonce() uint64
}

// .......................接口实现..................................//
type DefaultSortedTxs []*types.Transaction

// DefaultSortedTxs 相关方法全部改为指针接收者
func (sorted *DefaultSortedTxs) GasPrice() uint64 {
	if len(*sorted) == 0 {
		return 0
	}
	first := (*sorted)[0]
	return first.GasPrice
}

func (sorted *DefaultSortedTxs) Push(tx *types.Transaction) {
	*sorted = append(*sorted, tx)
}

// 从排序的交易中弹出，主要用在pending中
func (sorted *DefaultSortedTxs) Pop() *types.Transaction {
	if len(*sorted) == 0 {
		return nil
	}
	tx := (*sorted)[0]
	*sorted = (*sorted)[1:]
	return tx
}

func (sorted *DefaultSortedTxs) Replace(tx *types.Transaction) {
	for i, j := range *sorted {
		if j.Nonce == tx.Nonce {
			(*sorted)[i] = tx
			return
		}
	}
}

func (sorted *DefaultSortedTxs) FirstNonce() uint64 {
	if len(*sorted) == 0 {
		return 0
	}
	first := (*sorted)[0]
	return first.Nonce
}

func (sorted *DefaultSortedTxs) LastNonce() uint64 {
	if len(*sorted) == 0 {
		return 0
	}
	last := (*sorted)[len(*sorted)-1]
	return last.Nonce
}

// .........................用于按照gas排序..............................//
type pendingTxs []SortedTxs

func (p pendingTxs) Len() int {
	return len(p)
}
func (p pendingTxs) Less(i, j int) bool {
	return p[i].GasPrice() < p[j].GasPrice()
}

func (p pendingTxs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

//所以Sort.sort只能按照GasPrice排序，在replace中按照nonce排序要单独写
//................................交易池实现.................................................//

type DefaultPool struct {
	stat     statdb.StatDB
	all      map[hash.Hash]bool
	txs      pendingTxs
	pendings map[types.Address][]SortedTxs
	queue    map[types.Address][]*types.Transaction
}

func NewDefaultPool(stat statdb.StatDB) *DefaultPool {
	return &DefaultPool{
		stat:     stat,
		all:      make(map[hash.Hash]bool),
		txs:      make(pendingTxs, 0),
		pendings: make(map[types.Address][]SortedTxs),
		queue:    make(map[types.Address][]*types.Transaction),
	}
}
func (pool *DefaultPool) popFromQueue(addr *types.Address) *types.Transaction {
	list := pool.queue[*addr]
	if len(list) == 0 {
		return nil
	}
	tx := list[0]
	pool.queue[*addr] = list[1:]
	return tx
}

// PopPendingTx 从 pending 队列弹出一笔可打包的交易
func (pool *DefaultPool) PopPendingTx(addr types.Address) *types.Transaction {
	blks := pool.pendings[addr]
	if len(blks) == 0 {
		return nil
	}
	block := blks[0].(*DefaultSortedTxs)
	tx := block.Pop()
	if len(*block) == 0 {
		// 如果该 block 已空，移除
		pool.pendings[addr] = blks[1:]
	}
	return tx
}

// 过期交易清理（只保留 queue 中 nonce >= minNonce 且未过期的交易）
func (pool *DefaultPool) CleanExpiredTxs(minNonceMap map[types.Address]uint64) {
	const expireSec uint64 = 600 // 允许最大存活秒数，10分钟
	now := xtime.Now()           //返回当前时间
	for addr, txs := range pool.queue {
		minNonce := minNonceMap[addr]
		newTxs := make([]*types.Transaction, 0, len(txs))
		for _, tx := range txs {
			if tx.Nonce >= minNonce && now-tx.Time <= expireSec {
				newTxs = append(newTxs, tx)
			}
		}
		pool.queue[addr] = newTxs
	}
}

func (pool *DefaultPool) NewTx(tx *types.Transaction) {
	account := pool.stat.Load(tx.From())
	if account == nil {
		account = &types.Account{
			Nonce:  0,
			Amount: 0,
		}
	}
	if tx.Nonce < account.Nonce {
		return
	}

	nonce := account.Nonce
	blks := pool.pendings[tx.From()]
	if len(blks) > 0 {
		last := blks[len(blks)-1]
		nonce = last.LastNonce()
	}
	tx.Time = uint64(time.Now().Unix())
	if tx.Nonce > nonce+1 {
		pool.addQueue(tx)
	}
	if tx.Nonce == nonce+1 {
		pool.pushPendingTx(tx.From(), tx)
	}
	if tx.Nonce < nonce+1 {
		pool.replacePendingTx(pool.pendings[tx.From()], tx)
	}
}

func (pool *DefaultPool) addQueue(tx *types.Transaction) {
	list := pool.queue[tx.From()]
	replaced := false
	for i, old := range list {
		if old.Nonce == tx.Nonce {
			if tx.GasPrice > old.GasPrice {
				list[i] = tx // 替换为更高 gasPrice 的交易
			}
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, tx)
	}
	pool.queue[tx.From()] = list
	sort.Slice(pool.queue[tx.From()], func(i, j int) bool {
		return pool.queue[tx.From()][i].Nonce < pool.queue[tx.From()][j].Nonce
	})
}

func (pool *DefaultPool) pushPendingTx(addr types.Address, tx *types.Transaction) {
	blks := pool.pendings[addr]
	if len(blks) == 0 {
		pool.makeNewBlock(addr, tx)
	} else {
		last := blks[len(blks)-1]
		if last.GasPrice() <= tx.GasPrice {
			//[A1 A2 A3] [A4 A5]  A6(push into A5)
			//7   7   8    5  5	  5/6
			last.Push(tx)
		} else {
			pool.makeNewBlock(addr, tx)
		}
	}
	//pool.stat.IncNonce(addr) // 关键：每次有新交易进入pending都递增nonce
	pool.promoteQueueToPending(addr)
}

func (pool *DefaultPool) makeNewBlock(addr types.Address, tx *types.Transaction) {
	blk := make(DefaultSortedTxs, 0)
	blk = append(blk, tx)
	pool.pendings[addr] = append(pool.pendings[addr], &blk)
	pool.txs = append(pool.txs, &blk)
	sort.Sort(pool.txs)
}

// 从queue弹出交易加入pending
func (pool *DefaultPool) promoteQueueToPending(addr types.Address) {
	blks := pool.pendings[addr]
	if len(blks) == 0 {
		return
	}
	lastNonce := blks[len(blks)-1].LastNonce()
	queue := pool.queue[addr]
	for len(queue) > 0 && queue[0].Nonce == lastNonce+1 {
		tx := pool.popFromQueue(&addr)
		pool.pushPendingTx(addr, tx)
		blks = pool.pendings[addr]
		lastNonce = blks[len(blks)-1].LastNonce()
		queue = pool.queue[addr]
	}
}

func (pool *DefaultPool) replacePendingTx(blks []SortedTxs, tx *types.Transaction) {
	for _, blk := range blks {

		if blk.FirstNonce() <= tx.Nonce && blk.LastNonce() >= tx.Nonce && blk.GasPrice() <= tx.GasPrice {
			blk.Replace(tx)
			// replace
			//[A1 A2 A3][A4 A5]
			//blk1       blk2
			// 8  9   8   5   4
			// A4 
			// 6
			//还应该添加判断，如果换的是盒子里的第一个，看是否能加入前一个格子
			//比如新替换的A4 Gas>8，就应该加入blk1
			break
		}

	}
}

func (p *DefaultPool) NotifyTxEvent(txs []types.Transaction) {
	// 根据需要实现事件通知逻辑，或者留空
}

//....................供外部调用.........................//

func (p *DefaultPool) GetPendings() map[types.Address][]SortedTxs {
	return p.pendings
}
func (p *DefaultPool) GetQueue() map[types.Address][]*types.Transaction {
	return p.queue
}

func (p *DefaultPool) Pop() *types.Transaction {
	for addr, blks := range p.pendings {
		if len(blks) == 0 {
			continue
		}
		block := blks[0].(*DefaultSortedTxs)
		tx := block.Pop()
		if len(*block) == 0 {
			// 如果该 block 已空，移除
			p.pendings[addr] = blks[1:]
		}
		if tx != nil {
			return tx
		}
	}
	return nil
}
