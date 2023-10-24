package btc

import (
	"sync"

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
	return nil
}
