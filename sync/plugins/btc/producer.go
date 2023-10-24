package btc

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"gitlab.com/sync/common/config"
	"gitlab.com/sync/common/net/rpc"
	"gitlab.com/sync/features"
)

type producer struct {
	cfg    *config.Producer
	client *rpc.Client
}

func NewProducer(cfg *config.Producer) (features.Producer, error) {
	client, err := rpc.DialInsecureSkipVerify(cfg.URL, "", "", rpc.JSONRPCVersion2)
	if err != nil {
		return nil, err
	}
	p := &producer{
		cfg:    cfg,
		client: client,
	}
	return p, nil
}

func (p *producer) GetChainHeight() (int, error) {
	var height int
	if err := p.client.SyncCall(&height, getChainHeightMethod); err != nil {
		return 0, nil
	}
	return height, nil
}

func (p *producer) GetBlockByHeight(height int) (features.Block, error) {
	var hash string
	b := new(jsonBlock)
	err := p.client.SyncCall(&hash, getBlockHashMethod, height)
	if err != nil {
		return nil, err
	}
	err = p.client.SyncCall(&b, getBlockMethod, hash)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (p *producer) GetRelatedTransactions(block features.Block) ([]features.Transaction, error) {
	b := block.(*jsonBlock)
	var err error
	b.txes, err = p.batchTxes(b.Txes)
	if err != nil {
		return nil, err
	}
	err = p.convertTxes(b)
	if err != nil {
		return nil, err
	}
	var result []features.Transaction
	for _, v := range b.txes {
		result = append(result, v)
		for _, vin := range v.Vin {
			logrus.
				WithField("chain", "btc").
				WithField("transaction_hash", v.Hash).
				WithField("address", vin.address).
				WithField("amount", vin.value).
				WithField("index", vin.index).
				Info("input")
		}
		for _, vout := range v.Vout {
			logrus.
				WithField("chain", "btc").
				WithField("transaction_hash", v.Hash).
				WithField("address", vout.toAddress).
				WithField("amount", vout.value).
				WithField("index", vout.Index).
				Info("output")
		}
	}
	return result, nil
}

func (p *producer) batchTxes(hashes []string) ([]*jsonTransaction, error) {
	// TODO: btc每个块中交易数量太多，容易超时，为了演示只取前10条
	var max int
	if len(hashes) > 10 {
		max = 10
	} else {
		max = len(hashes)
	}
	request := make([]rpc.BatchElem, 0, max)
	txes := make([]*jsonTransaction, max)
	for i, hash := range hashes[:max] {
		txes[i] = new(jsonTransaction)
		request = append(request, rpc.BatchElem{Method: getRawTransactionMethod, Args: []interface{}{hash, 1}, Result: txes[i]})
	}
	err := p.client.BatchSyncCall(request)
	if err != nil {
		return nil, err
	}
	for _, elem := range request {
		if elem.Error != nil {
			return nil, elem.Error
		}
	}
	for index, hash := range hashes[:max] {
		if hash != txes[index].Hash {
			err = fmt.Errorf("bacth tx err: %d/%d wanted %s, but got %s",
				index+1, len(hashes), hash, txes[index].Hash)
			return nil, err
		}
	}
	return txes, nil
}

func (p *producer) convertTxes(b *jsonBlock) error {
	err := b.convertWithoutCheckNode()
	if err != nil {
		return err
	}
	txes, err := p.batchTxes(b.getUncheckedVinAddressPreHashes())
	if err != nil {
		return err
	}

	err = b.convertWithCheckNode(txes)
	if err != nil {
		return err
	}
	txes, err = p.batchTxes(b.getUncheckedVinValuePreHashes())
	if err != nil {
		return err
	}
	return b.fillFromAddress()
}
