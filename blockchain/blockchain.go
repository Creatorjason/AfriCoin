package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First block in chain"
	
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}
type BlockchainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func DBexist(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}
func InitializeBlockchain(address, nodeId string) *BlockChain {
	var lastHash []byte
	path := fmt.Sprintf(dbPath, nodeId)
	if DBexist(path) {
		fmt.Printf("Blockchain exist\n")
		runtime.Goexit()
	}
	opts := badger.DefaultOptions(dbPath)
	db, err := openDB(path, opts)
	HandleErr(err)
	err = db.Update(func(txn *badger.Txn) error {
		coinbaseTrx := CoinbaseTx(address, genesisData)
		genesis := CreateGenesisBlock(coinbaseTrx)
		err = txn.Set(genesis.Hash, genesis.Serialize())
		HandleErr(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err

	})
	HandleErr(err)
	blockchain := &BlockChain{lastHash, db}
	return blockchain
}

func openDB(dir string, opts badger.Options) (*badger.DB, error){
	if db, err := badger.Open(opts); err != nil{
		if strings.Contains(err.Error(), "LOCK"){
			if db, err := retry(dir, opts); err != nil{
				log.Println("database unlocked, value log truncated")
				return db, nil
			} 
			log.Println("could not unlock database", err)
		}
		return nil, err
	}else{
		return db, nil
	} 
}

func retry(dir string, origOpts badger.Options) (*badger.DB, error){
	lockPath := filepath.Join(dir,"LOCK")
	if err := os.Remove(lockPath); err != nil{
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := origOpts
	// retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	
	return db, err
}

func ContinueBlockchain(nodeId string) *BlockChain {
	var lastHash []byte
	path := fmt.Sprintf(dbPath, nodeId)
	if !DBexist(path) {
		fmt.Printf("No Blockchain found\n")
		runtime.Goexit()
	}
	opts := badger.DefaultOptions(dbPath)
	db, err := openDB(path, opts)
	HandleErr(err)
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		return err
	})
	HandleErr(err)
	blockchain := &BlockChain{lastHash, db}
	return blockchain

}

func (chain *BlockChain) AddBlock(block *Block){
	var lastHash []byte
	var lastBlock *Block
	err := chain.Database.Update(func(txn *badger.Txn) error{
		if _, err := txn.Get(block.Hash); err == nil{
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		HandleErr(err)
	

		item, err := txn.Get([]byte("lh"))
		HandleErr(err)
		_ = item.Value(func(val []byte) error{
			lastHash = val
			return  nil
		})
		// HandleErr(err)
		item, err = txn.Get(lastHash)
		HandleErr(err)
		err = item.Value(func(val []byte) error{
			lastBlock = Deserialize(val)
			return nil
		})
		HandleErr(err)
		if block.Height > lastBlock.Height{
			err = txn.Set([]byte("lh"), block.Hash)
			HandleErr(err)
			chain.LastHash = block.Hash
		}
		return nil

	})
	HandleErr(err)
}

func (chain *BlockChain)MineBlock(transaction []*Transaction) *Block{
	var lastHash []byte
	var lastHeight int
	var lastBlock *Block

	for _, tx := range transaction{
		if !chain.VerifyTx(tx){
			panic("Invalid transaction")
		}
	}

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		item, err = txn.Get(lastHash)
		HandleErr(err)
		err = item.Value(func(val []byte)error{
			lastBlock = Deserialize(val)
			return nil
		})
		lastHeight = lastBlock.Height
		return err
	})
	HandleErr(err)

	newBlock := CreateBlock(transaction, lastHash, lastHeight+1)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		HandleErr(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	HandleErr(err)
	return newBlock
}

func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error){
	var block Block
	// var blockData []byte
	
	err := chain.Database.View(func(txn *badger.Txn) error{
		if item, err := txn.Get(blockHash); err != nil{
			return errors.New("no blocks found")
		}else{
			_ = item.Value(func(val []byte) error {
				
				block = *Deserialize(val)
				return nil
			})
		}
		return nil
	})
	if err != nil{
		return block, err
	}
	HandleErr(err)
	return block, nil
}

func (chain *BlockChain) GetBlocksHashes() [][]byte{
	var blocks [][]byte

	iter := chain.Iterator()
	for{
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0{
			break
		}

	}
	return blocks
}

func (chain *BlockChain) GetBestHeight() int{
	var lastBlock Block
	var lastHash []byte
	err := chain.Database.View(func(txn *badger.Txn) error{
		item, err := txn.Get([]byte("lh"))
		HandleErr(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		item, err = txn.Get(lastHash)
		HandleErr(err)
		err = item.Value(func(val []byte)error{
			lastBlock = *Deserialize(val)
			return nil
		})
		return err
	})
	HandleErr(err)
	return lastBlock.Height
}

func (chain *BlockChain) FindUTXO() map[string]OutputsArr{
	// var unspentTxs []Transaction
	UTXO := make(map[string]OutputsArr)
	spentTXOs := make(map[string][]int)
	iter := chain.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if !tx.IsCoinbaseTxn() {
				for _, in := range tx.Vin {
						inTxid := hex.EncodeToString(in.TXID)
						spentTXOs[inTxid] = append(spentTXOs[inTxid], in.Vout)
					
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXO
}


func (chain *BlockChain) FindTrxById(ID []byte) (Transaction, error) {
	iter := chain.Iterator()
	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist ")
}

func (chain *BlockChain) SignTrx(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Vin {
		prevTx, err := chain.FindTrxById(in.TXID)
		HandleErr(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	tx.Sign(privKey, prevTxs)
}

func (chain *BlockChain) VerifyTx(tx *Transaction) bool {
	if tx.IsCoinbaseTxn(){
		return true
	}
	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Vin {
		prevTx, err := chain.FindTrxById(in.TXID)
		HandleErr(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	return tx.Verify(prevTxs)
}

func DeserializeTrx(data []byte) Transaction{
	var transaction Transaction
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	HandleErr(err)
	return transaction
}