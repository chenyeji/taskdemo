package eth

import (
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/sync/common"
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
	var res string
	if err := p.client.SyncCallObject(&res, getBlockNumber, []bool{}); err != nil {
		return 0, err
	}
	height, err := common.DecodeHex(res)
	if err != nil {
		return 0, err
	}
	return int(height), nil
}

func (p *producer) GetBlockByHeight(height int) (features.Block, error) {
	b := new(jsonBlock)
	err := p.client.SyncCall(b, getBlock, fmt.Sprintf("0x%x", height), true)
	if err != nil {
		return nil, err
	}

	if err := b.convert(); err != nil {
		return nil, err
	}
	return b, nil
}

func (p *producer) GetRelatedTransactions(block features.Block) ([]features.Transaction, error) {
	var err error
	b := block.(*jsonBlock)
	if !b.receiptsReady {
		for _, tx := range b.Transactions {
			err := tx.convert()
			if err != nil {
				return nil, err
			}
		}
		hashes := make([]string, 0, len(b.Transactions))
		for _, item := range b.Transactions {
			hashes = append(hashes, item.Hash)
		}
		receipts, err := batchTransactionReceiptBatchSearch(p.client, hashes)
		if err != nil {
			return nil, err
		}
		if len(b.Transactions) != len(receipts) {
			return nil, fmt.Errorf("the receipts number: [%d] is not match related txes's: [%d] in %d",
				len(receipts), len(b.Transactions), b.height)
		}
		b.receiptsMap = make(map[string]*jsonTransactionReceipt)
		for index, receipt := range receipts {
			if len(receipt.BlockHash) == 0 {
				return nil, errors.Errorf("block %d %s. receipt's block hash is is invalid data,txhash %s", b.height, b.Hash, b.Transactions[index].Hash)
			}
			if receipt.BlockHash != b.Hash {
				return nil, errors.Errorf("block %d %s. receipt's block hash is %s,txhash %s", b.height, b.Hash, receipt.BlockHash, receipt.TransactionHash)
			}
			b.receiptsMap[receipt.TransactionHash] = &receipts[index]
		}
		b.receiptsReady = true
	}
	result := make([]features.Transaction, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		receipt, ok := b.receiptsMap[tx.hash]
		if !ok {
			return nil, fmt.Errorf("tx %s not find match receipt", tx.hash)
		}
		err = tx.combineReceipt(receipt)
		if err != nil {
			return nil, err
		}
		if tx.receiptStatusSuccess() {
			tx.tokenTfs, err = p.getTokenTransfer(tx, receipt.Logs)
			if err != nil {
				return nil, err
			}
		}
		if len(tx.tokenTfs) > 0 {
			result = append(result, tx)
		}
	}
	return result, nil
}

func (p *producer) getTokenTransfer(rTx *jsonTransaction, logs []receiptLogs) ([]*transfer, error) {
	result := make([]*transfer, 0)
	// filter by transaction log event
	for _, tLog := range logs {
		if len(tLog.Topics) == 3 && tLog.Topics[0] == transferEventHash {
			result = append(result, &transfer{
				txHash:         rTx.hash,
				assetChainName: tLog.Address,
				from:           tLog.Topics[1],
				to:             tLog.Topics[2],
				amount:         tLog.Data,
				index:          tLog.LogIndex,
				tradeType:      contractTradeType,
			})
		}
	}
	return result, nil
}

func batchTransactionReceiptBatchSearch(client *rpc.Client, hashes []string) ([]jsonTransactionReceipt, error) {
	batchList := make([]rpc.BatchElem, 0, len(hashes))
	receiptList := make([]jsonTransactionReceipt, len(hashes))
	if len(hashes) == 0 {
		return receiptList, nil
	}
	for i, hash := range hashes {
		batchList = append(batchList, rpc.BatchElem{
			Method: getReceipt,
			Args:   []interface{}{hash},
			Result: &receiptList[i]})
	}
	err := client.BatchSyncCall(batchList)
	if err != nil {
		temp := hashes
		if len(temp) > 10 {
			temp = temp[:10]
		}
		return nil, errors.Wrapf(err, "%s %d, %v", getReceipt, len(hashes), temp)
	}

	// check elem error
	for _, elem := range batchList {
		err = elem.Error
		if err != nil {
			return nil, err
		}
	}
	return receiptList, nil
}
