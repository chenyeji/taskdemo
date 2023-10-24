package eth

import (
	"sync"

	"github.com/sirupsen/logrus"

	"gitlab.com/sync/common/config"
	"gitlab.com/sync/features"
)

type consumer struct {
	sync.RWMutex
	currentHeight int
}

func NewConsumer(cfg *config.Consumer) (features.Consumer, error) {
	return &consumer{
		currentHeight: cfg.StartHeight,
	}, nil
}

func (c *consumer) GetCurrentBlockInfo() (features.Block, error) {
	c.Lock()
	defer c.Unlock()
	return &features.BlockInfo{
		Height: c.currentHeight,
	}, nil
}

func (c *consumer) NewBlock(b features.Block, txs []features.Transaction) error {
	c.Lock()
	defer c.Unlock()
	c.currentHeight = b.GetHeight()
	for _, tx := range txs {
		logrus.
			WithField("chain", "eth").
			WithField("transaction_hash", tx.GetHash()).
			WithField("token_address", tx.ToAddress()).
			WithField("from_address", tx.FromAddress()).
			WithField("to_address", tx.ToAddress()).
			WithField("amount", tx.Amount()).
			Infof("erc20 transactions on block %d", b.GetHeight())
	}
	return nil
}
