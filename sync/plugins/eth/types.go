package eth

import (
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	"gitlab.com/sync/common"
)

const (
	getBlockNumber = "eth_blockNumber"
	getBlock       = "eth_getBlockByNumber"
	getReceipt     = "eth_getTransactionReceipt"

	transferEventHash = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

type jsonBlock struct {
	BaseFeePerGas   string             `json:"baseFeePerGas"` //EIP-1559
	Number          string             `json:"number"`
	Hash            string             `json:"hash"`
	ParentHash      string             `json:"parentHash"`
	Miner           string             `json:"miner"`
	Timestamp       string             `json:"timestamp"`
	TotalDifficulty string             `json:"totalDifficulty"`
	Difficulty      string             `json:"difficulty"`
	ExtraData       string             `json:"extraData"`
	Nonce           string             `json:"nonce"`
	Uncles          []string           `json:"uncles"`
	Transactions    []*jsonTransaction `json:"transactions"`

	receiptsReady bool
	receiptsMap   map[string]*jsonTransactionReceipt

	height, time     int
	hash, parentHash string
	uncles           []string

	baseFeePerGas *big.Int //EIP-1559
}

func (b *jsonBlock) GetHash() string {
	return b.hash
}

func (b *jsonBlock) GetHeight() int {
	return b.height
}

func (b *jsonBlock) GetParentHash() string {
	return b.parentHash
}

func (b *jsonBlock) GetBlockTime() int {
	return b.time
}

type jsonTransactionReceipt struct {
	TransactionHash   string        `json:"transactionHash"`
	TransactionIndex  string        `json:"transactionIndex"`
	BlockNumber       string        `json:"blockNumber"`
	BlockHash         string        `json:"blockHash"`
	CumulativeGasUsed string        `json:"cumulativeGasUsed"`
	EffectiveGasPrice string        `json:"effectiveGasPrice"` //EIP-1559 price = min(maxPriorityFeePerGas, maxFeePerGas - baseFee) + baseFee
	GasUsed           string        `json:"gasUsed"`
	Status            string        `json:"status"`
	Logs              []receiptLogs `json:"logs"`

	// opteth
	// L2 execution fee = tx.gasPrice * l2GasUsed
	// 总手续费 = L2 execution fee  + L1 security fee
	L1Fee string `json:"l1Fee"`
}

type receiptLogs struct {
	LogIndex         string   `json:"logIndex"`
	TransactionIndex string   `json:"transactionIndex"`
	TransactionHash  string   `json:"transactionHash"`
	BlockHash        string   `json:"blockHash"`
	BlockNumber      string   `json:"blockNumber"`
	Address          string   `json:"address"`
	Data             string   `json:"data"`
	Topics           []string `json:"topics"`
	Type             string   `json:"type"`
}

func (b *jsonBlock) convert() error {
	value, err := common.GetHexNumber(b.Number)
	if err != nil {
		return err
	}
	uTime, err := strconv.ParseUint(b.Timestamp, 0, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	b.time = int(uTime)
	b.height = int(value.Int64())
	b.hash = b.Hash
	b.parentHash = b.ParentHash
	b.uncles = make([]string, 0, len(b.Uncles))
	for _, item := range b.Uncles {
		b.uncles = append(b.uncles, item)
	}
	return nil
}

type jsonTransaction struct {
	Hash                 string `json:"hash"`
	Nonce                string `json:"nonce"`
	BlockHash            string `json:"blockHash"`
	BlockNumber          string `json:"blockNumber"`
	TransactionIndex     string `json:"transactionIndex"`
	From                 string `json:"from"`
	To                   string `json:"to"`
	Value                string `json:"value"`
	Gas                  string `json:"gas"`
	GasPrice             string `json:"gasPrice"`             //EIP-1559 =jsonTransactionReceipt.effectiveGasPrice
	MaxFeePerGas         string `json:"maxFeePerGas"`         //EIP-1559
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"` //EIP-1559
	Input                string `json:"input"`
	Asset                string `json:"asset"`      //for eminer asset
	Action               int    `json:"action"`     //for eminer transfer action
	SubAddress           string `json:"subAddress"` //for eminer transfer subAddress
	TxType               string `json:"type"`

	hash, from, to, totag                             string
	tfs                                               []*transfer
	tokenTfs                                          []*transfer
	gas, gasPrice, gasUsed, amount, fee               *big.Int
	nonce, status, receiptStatus, blockHeight, txType uint64
}

func (t *jsonTransaction) GetHash() string {
	return t.Hash
}

func (t *jsonTransaction) TokenAddress() string {
	if len(t.tokenTfs) == 0 {
		return ""
	}
	return t.tokenTfs[0].assetChainName
}
func (t *jsonTransaction) FromAddress() string {
	return t.From
}

func (t *jsonTransaction) ToAddress() string {
	return t.To
}

func (t *jsonTransaction) Amount() string {
	if len(t.tokenTfs) == 0 {
		return ""
	}
	return t.tokenTfs[0].amount
}

func (t *jsonTransaction) receiptStatusSuccess() bool {
	return t.receiptStatus == 1
}

func (t *jsonTransaction) convert() error {
	t.hash = t.Hash
	var err error
	t.txType, err = strconv.ParseUint(t.TxType, 0, 64)
	if err != nil {
		return err
	}
	t.from = t.From
	t.to = t.To

	t.tfs = make([]*transfer, 0, 1)
	t.tokenTfs = make([]*transfer, 0, 1)
	t.nonce, err = strconv.ParseUint(t.Nonce, 0, 64)
	return errors.WithStack(err)
}

func (t *jsonTransaction) combineReceipt(r *jsonTransactionReceipt) error {
	var err error
	t.receiptStatus, err = strconv.ParseUint(r.Status, 0, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	t.status = t.receiptStatus
	return nil
}

type transfer struct {
	from, to, totag string
	amount          string
	index           string
	txHash          string
	fee             string
	tradeType       tradeType
	assetChainName  string
}

type tradeType string

const (
	coinbaseTradeType          tradeType = "coinbase"
	transferTradeType          tradeType = "transfer"
	contractTradeType          tradeType = "contract"
	nftTransferTradeType       tradeType = "nft_transfer"
	nftTransferSingleTradeType tradeType = "nft_transfer_single"
	nftTransferBatchTradeType  tradeType = "nft_transfer_batch"
	mainCoinContractTradeType  tradeType = "main_coin_contract"
	celoGoldTokenProxy                   = "471ece3750da237f93b8e339c536989b8978a438"
	maticMRC20                           = "0000000000000000000000000000000000001010"
)
