package blockchain

import (
	// "crypto/sha256"
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

// Module describing Blocks

type Block struct {
	PrevHash     []byte
	Transactions []*Transaction
	Hash         []byte
	Nonce        int
	Timestamp    int64
	Height       int
}

// Method to derive hash of block
// func (b *Block) DeriveHash() []byte{
// 	data := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
// 	hash := sha256.Sum256(data)

// 	return hash[:]
// }

func CreateBlock(txs []*Transaction, prevHash []byte, height int) *Block {
	block := &Block{prevHash, txs, []byte{}, 0, time.Now().Unix(), height}
	pow := ComputeTargetForBlock(block)
	nonce, hash := pow.RunPOW()
	block.Nonce = nonce
	block.Hash = hash
	return block
}

func CreateGenesisBlock(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
}

func (block *Block) HashTransactions() []byte{
	var txHashes [][]byte

	allTx := block.Transactions
	for _, tx := range allTx{
		txHashes = append(txHashes, tx.HashTx())
	}
	// data := bytes.Join(txHashes, []byte{})
	// txHash = sha256.Sum256(data ) 
	tree := NewMerkleTree(txHashes)
	return tree.RootNode.Data
}

func (b *Block) Serialize() []byte {
	// to uses gobs, create an encoder and present it with a series of data
	// Create bytes buffer, for storing data streams
	buff := new(bytes.Buffer)

	// Create an encoder, to encode buffer containing bytes streams for transfer
	encoder := gob.NewEncoder(buff)

	//  Encode the data to the sent, also transmits the data
	err := encoder.Encode(b)
	HandleErr(err)

	return buff.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)

	HandleErr(err)

	return &block
}

func HandleErr(err error) {
	if err != nil {
		log.Panic(err)
	}
}
