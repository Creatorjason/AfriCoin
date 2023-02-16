package blockchain

import (
	badger "github.com/dgraph-io/badger/v3"
)

func (chain *BlockChain) Iterator() *BlockchainIterator {
	iter := &BlockchainIterator{chain.LastHash, chain.Database}

	return iter
}

func (itr *BlockchainIterator) Next() *Block {
	var block *Block
	err := itr.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(itr.CurrentHash)
		HandleErr(err)
		// get the block, deserialize it and return it
		err = item.Value(func(val []byte) error {
			block = Deserialize(val)
			return nil
		})
		return err
	})
	HandleErr(err)
	// update hash value
	itr.CurrentHash = block.PrevHash
	return block
}
