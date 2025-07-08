package statemachine

import (
	"cxchain-2023131076/statdb"
	"cxchain-2023131076/types"
	"fmt"
)

type IMachine interface {
	Execute(state statdb.StatDB, tx types.Transaction) *types.Receiption
}

type StateMachine struct{}

func (m StateMachine) Execute(state statdb.StatDB, tx types.Transaction) *types.Receiption {
	from := tx.From()
	to := tx.To
	value := tx.Value

	// gasUsed 逻辑修正
	gasUsed := tx.Gas
	if gasUsed < 21000 {
		return &types.Receiption{Status: 0, GasUsed: gasUsed}
	}
	gasUsed = 21000
	gasCost := gasUsed * tx.GasPrice
	cost := value + gasCost

	account := state.Load(from)
	fmt.Printf("[DEBUG] Execute: from=%x, to=%x, value=%d, cost=%d, account=%+v\n", from, to, value, cost, account)
	if account == nil || account.Amount < cost {
		return &types.Receiption{Status: 0, GasUsed: gasUsed}
	}

	account.Amount -= cost
	account.Nonce += 1
	state.Store(from, *account)

	toAccount := state.Load(to)
	if toAccount == nil {
		toAccount = &types.Account{}
	}
	toAccount.Amount += value
	state.Store(to, *toAccount)

	return &types.Receiption{Status: 1, GasUsed: gasUsed}
}
