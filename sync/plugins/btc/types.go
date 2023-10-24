package btc

import (
	"encoding/json"
	"math/big"

	"gitlab.com/sync/common"

	"github.com/pkg/errors"
)

const (
	getChainHeightMethod    = "getblockcount"
	getBlockHashMethod      = "getblockhash"
	getBlockMethod          = "getblock"
	getRawTransactionMethod = "getrawtransaction"
)

type jsonBlock struct {
	Hash          string   `json:"hash"`
	Height        int      `json:"height"`
	Time          int      `json:"time"`
	PrevBlockHash string   `json:"previousblockhash"`
	Txes          []string `json:"tx"`
	PosFlag       string   `json:"flags"`
	NTx           int      `json:"nTx"` //ltc

	txes  []*jsonTransaction
	miner string

	Chainlock bool `json:"chainlock"` // SYSCOIN 独有的，用于判断区块是否不可篡改

}

func (b *jsonBlock) GetHash() string {
	return b.Hash
}

func (b *jsonBlock) GetHeight() int {
	return b.Height
}

func (b *jsonBlock) GetParentHash() string {
	return b.PrevBlockHash
}

func (b *jsonBlock) GetBlockTime() int {
	return b.Time
}

func (b *jsonBlock) convertWithoutCheckNode() error {
	for _, tx := range b.txes {
		err := tx.convert(b.Height, false)
		if err != nil {
			return errors.Wrapf(err, "tx %s", tx.Hash)
		}
	}
	return nil
}

func (b *jsonBlock) convertWithCheckNode(txes []*jsonTransaction) error {
	txMap := make(map[string]*jsonTransaction, len(txes))
	for _, tx := range txes {
		txMap[tx.Hash] = tx
	}
	for _, tx := range b.txes {
		for _, vin := range tx.Vin {
			if vin.gotAddress {
				continue
			}
			preTx, ok := txMap[vin.PrevTxHash]
			if !ok {
				return errors.Errorf("lost pre tx(%s) while check %s's vin(%d)",
					vin.PrevTxHash, tx.Hash, vin.index)
			}
			if int64(len(preTx.Vout)) < vin.PrevVoutIndex+1 {
				return errors.Errorf("lost pre tx(%s)'s vout(%d) while check %s's vin(%d)",
					vin.PrevTxHash, vin.PrevVoutIndex, tx.Hash, vin.index)
			}
			vout := preTx.Vout[vin.PrevVoutIndex]
			err := vout.convert()
			if err != nil {
				return err
			}
			vin.address = vout.toAddress
			vin.value = vout.value
			vin.gotAddress = true
			vin.gotValue = true
		}
	}
	return nil
}

func (b *jsonBlock) getUncheckedVinAddressPreHashes() (hashes []string) {
	hashMap := make(map[string]bool, len(b.txes))
	for _, tx := range b.txes {
		for _, vin := range tx.Vin {
			if !vin.gotAddress {
				hashMap[vin.PrevTxHash] = true
			}
		}
	}
	hashes = make([]string, 0, len(hashMap))
	for hash := range hashMap {
		hashes = append(hashes, hash)
	}
	return
}

func (b *jsonBlock) getUncheckedVinValuePreHashes() (hashes []string) {
	hashMap := make(map[string]bool, len(b.txes))
	for _, tx := range b.txes {
		for _, vin := range tx.Vin {
			if !vin.gotValue {
				hashMap[vin.PrevTxHash] = true
			}
		}
	}
	hashes = make([]string, 0, len(hashMap))
	for hash := range hashMap {
		hashes = append(hashes, hash)
	}
	return
}

func (b *jsonBlock) fillFromAddress() error {
	for _, tx := range b.txes {
		if err := tx.fillFromAddress(); err != nil {
			return err
		}
	}
	return nil
}

type jsonTransaction struct {
	Hash      string      `json:"txid"`
	BlockHash string      `json:"blockhash"`
	Time      int         `json:"time"`
	BlockTime int         `json:"blocktime"`
	Vin       []*jsonVin  `json:"vin"`
	Vout      []*jsonVout `json:"vout"`

	// vShieldedSpend+vShieldedOutput 用来判断是否为匿名交易
	//VShieldedSpend  []json.RawMessage `json:"vShieldedSpend"`
	//VShieldedOutput []json.RawMessage `json:"vShieldedOutput"`
	vinRelated bool
	shielded   bool // set for zec。true 表示这是匿名交易

	txType string
	fee    *big.Int

	writeDeposit bool // only work for wallet and bcd, for airdrop
}

func (t *jsonTransaction) GetHash() string {
	return t.Hash
}

func (t *jsonTransaction) TokenAddress() string {
	return ""
}

func (t *jsonTransaction) FromAddress() string {
	return ""
}

func (t *jsonTransaction) ToAddress() string {
	return ""
}

func (t *jsonTransaction) Amount() string {
	return ""
}

func (t *jsonTransaction) convert(height int, isTest bool) error {
	if t.isCoinbase() {
		t.txType = "1"
		t.Vin = []*jsonVin{}
	}

	for i, vin := range t.Vin {
		vin.convertAddressWithoutCheckNode(isTest, int64(i))
	}

	for _, vout := range t.Vout {
		if err := vout.convert(); err != nil {
			return err
		}
	}
	return nil
}

func (t *jsonTransaction) isCoinbase() bool {
	if len(t.Vin) == 1 {
		if len(t.Vin[0].CoinBase) > 0 {
			return true
		}
	}
	return false
}

func (t *jsonTransaction) fillFromAddress() error {
	result := []*jsonVout{}
	for _, vout := range t.Vout {
		for _, vin := range t.Vin {
			vout.value = vin.value
			vout.fromAddress = vin.address
			result = append(result, vout)
		}
	}
	t.Vout = result
	return nil
}

type scriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type jsonVin struct {
	PrevTxHash    string    `json:"txid"`
	PrevVoutIndex int64     `json:"vout"`
	CoinBase      string    `json:"coinbase"`
	CoinBase2     string    `json:"coinbase2"` // set for sbtc
	Sequence      int       `json:"sequence"`
	ScriptSig     scriptSig `json:"scriptSig"`
	ValueSat      uint64    `json:"valueSat"` // set for xzc, look for tx: 378bb77b494a5ed9a07250b58598418006f6869aae176d9ae045579cb6b6f97b
	Ismweb        bool      `json:"ismweb"`   //ltc

	index   int64
	address string
	value   string

	gotAddress bool
	gotValue   bool
}

func (v *jsonVin) convertAddressWithoutCheckNode(isTest bool, index int64) {
	v.index = index
	var err error
	v.address, err = common.GetAddressFromScriptSig(v.ScriptSig.Asm, isTest)
	v.gotAddress = (err == nil)
}

type jsonScriptPubKey struct {
	Address   string   `json:"address"` //v22+版本btc程序新增
	Addresses []string `json:"addresses"`
	Type      string   `json:"type"`
}

type jsonVout struct {
	Value        json.RawMessage  `json:"value"`
	Index        int64            `json:"n"`
	ScriptPubKey jsonScriptPubKey `json:"scriptPubKey"`
	Ismweb       bool             `json:"ismweb"` //ltc

	isConverted bool
	fromAddress string
	toAddress   string
	value       string
	UUID        string
}

func (v *jsonVout) convert() error {
	// address
	var err error
	v.toAddress, err = getAddressFromScript(v.ScriptPubKey)
	if err != nil {
		return err
	}

	// value
	value, err := getSatoshiValue(v.Value)
	if err != nil {
		return err
	}
	v.value = value
	v.isConverted = true
	return nil
}

func getAddressFromScript(script jsonScriptPubKey) (string, error) {
	return script.Address, nil
}

func getSatoshiValue(value json.RawMessage) (string, error) {
	return string(value), nil
}
