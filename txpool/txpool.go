package txpool

import(
	"cxchain-2023131076/types"
	// "cxchain-2023131076/utils/hash"
)

type TxPool interface{
	//SetStatRoot(root hash.Hash),更新状态树的根哈希,在stat.go里	NewTx(tx *types.Transaction)//接收新交易
	Pop() *types.Transaction//从pending中pop出
	NotifyTxEvent(txs []types.Transaction)  //通知新交易
}
