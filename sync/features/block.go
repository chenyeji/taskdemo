package features

type Block interface {
	GetHash() string
	GetHeight() int
	GetParentHash() string
	GetBlockTime() int
}

type BlockInfo struct {
	Hash       string
	Height     int
	ParentHash string
	BlockTime  int
}

func (b *BlockInfo) GetHash() string {
	return b.Hash
}

func (b *BlockInfo) GetHeight() int {
	return b.Height
}

func (b *BlockInfo) GetParentHash() string {
	return b.ParentHash
}

func (b *BlockInfo) GetBlockTime() int {
	return b.BlockTime
}
