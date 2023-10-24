package core

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/sync/common"
	"gitlab.com/sync/common/config"
	"gitlab.com/sync/features"
	"gitlab.com/sync/plugins"
)

const minDuration time.Duration = -1 << 63

type Processor struct {
	*config.Config
	plugins map[string]*plugins.Plugin
}

func NewProcessor(c *config.Config) (features.Processor, error) {
	if p, err := plugins.Loader(c.App.Chains, c); err != nil {
		return nil, err
	} else {
		return &Processor{
			Config:  c,
			plugins: p,
		}, nil
	}
}

func (p *Processor) Loop(shutdown chan struct{}) {
	var wg sync.WaitGroup
	for k, v := range p.plugins {
		wg.Add(1)
		go func(chain string, producer features.Producer, consumer features.Consumer) {
			defer wg.Done()
			timer := time.NewTimer(minDuration)
			for {
				select {
				case <-shutdown:
					logrus.Infof("%s stop working", chain)
					return
				case <-timer.C:
					emptyLoop, err := p.worker(chain, producer, consumer)
					if err != nil {
						logrus.Error(err)
						timer.Reset(time.Millisecond * time.Duration(p.App.ErrorInterval))
					} else if emptyLoop {
						timer.Reset(time.Millisecond * time.Duration(p.App.EmptyInterval))
					} else {
						timer.Reset(minDuration)
					}
				}
			}
		}(k, v.Producer, v.Consumer)
	}
	wg.Wait()
}

func (p *Processor) worker(chain string, producer features.Producer, consumer features.Consumer) (bool, error) {
	logrus.
		WithField("chain", chain).
		Infof("worker start")
	start := time.Now()
	defer common.TimeConsume(start)

	current, err := consumer.GetCurrentBlockInfo()
	if err != nil {
		return false, err
	}

	lastBlockHeight := current.GetHeight()
	nextBlockHeight := lastBlockHeight + 1
	maxBlockHeight, err := producer.GetChainHeight()
	if err != nil {
		return false, err
	}
	if nextBlockHeight > maxBlockHeight && maxBlockHeight > 0 {
		logrus.
			WithField("chain", chain).
			WithField("next_block_height", nextBlockHeight).
			WithField("max_block_height", maxBlockHeight).
			Infof("reach max block height")
		return true, nil
	}
	nextBlock, err := producer.GetBlockByHeight(nextBlockHeight)
	if err != nil {
		return false, err
	}
	txs, err := producer.GetRelatedTransactions(nextBlock)
	if err != nil {
		return false, err
	}
	if err := consumer.NewBlock(nextBlock, txs); err != nil {
		return false, err
	}

	logrus.
		WithField("chain", chain).
		WithField("block_height", 0).
		WithField("cost", time.Since(start).String()).
		Info("worker complete")
	return false, nil
}
