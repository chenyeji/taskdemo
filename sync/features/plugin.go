package features

// Producer Fetch on chain data
type Producer interface {
	GetChainHeight() (int, error)
	GetBlockByHeight(height int) (Block, error)
	GetRelatedTransactions(b Block) ([]Transaction, error)
}

// Consumer ...
type Consumer interface {
	GetCurrentBlockInfo() (Block, error)
	NewBlock(block Block, txs []Transaction) error
}

type Transaction interface {
	GetHash() string
	TokenAddress() string
	FromAddress() string
	ToAddress() string
	Amount() string
}
