package blockchain

import (
	"encoding/hex"
	"github.com/dgraph-io/badger/v3"
	"bytes"
)

type UTXOset struct {
	Blockchain *BlockChain
}

// KV stores doesn't have tables that helps group and separate data
// Using a prefix helps us introduces grouping and differentiation to data

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

func (set *UTXOset) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := set.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 1000000
	set.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()
		keysForDelete := make([][]byte, 0, collectSize)
		// the size of the two dimensional slice is determined ny the value given: 0 for the first slice, capacity for the underlying array backing the slice
		keysCollected := 0
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			key := iter.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}
		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				panic(err)
			}
		}
		return nil
	})

}

func (utxo UTXOset) Reindex() {
	db := utxo.Blockchain.Database
	utxo.DeleteByPrefix(utxoPrefix)
	UTXO := utxo.Blockchain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			HandleErr(err)
			key = append(utxoPrefix, key...)
			err = txn.Set(key, outs.SerializeOutputs())
			HandleErr(err)
		}
		return nil
	})
	HandleErr(err)
}
func (u UTXOset) FindUTXO(pubKeyHash []byte) []TxOutputs {
	var UTXOs []TxOutputs
	var value []byte

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error{
				value = val
				return nil
			})
			HandleErr(err)
			outs := DeserializeOutputs(value)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	HandleErr(err)

	return UTXOs
}
func (u UTXOset) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	var value []byte
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(val []byte) error{
				value = val
				return nil
			})
			HandleErr(err)
			
			k = bytes.TrimPrefix(k, utxoPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(value)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}
		return nil
	})
	HandleErr(err)
	return accumulated, unspentOuts
}

// func (utxo *UTXOset) UpdateSet(block *Block){
// 	var value []byte

// 	db := utxo.Blockchain.Database

// 	err := db.Update(func (txn *badger.Txn) error{
// 		for _, tx := range block.Transactions{
// 			if !tx.IsCoinbaseTxn(){
// 				for _, in := range tx.Vin{
// 					updatedOuts := OutputsArr{}
// 					inID := append(utxoPrefix, in.TXID...)
// 					item, err := txn.Get(inID)
// 					HandleErr(err)
// 					err = item.Value(func (val []byte) error{
// 						value = val
// 						return nil
// 					})
// 					HandleErr(err)
// 						}
// 					}
// 				}
// 			}

// 		}
// 		return nil
// 	})
// 	HandleErr(err)
// }

// outs := DeserializeOutputs(value)
// for outIdx, out := range outs.Outputs{
// if outIdx != in.Vout{
// updatedOuts.Outputs = append(updatedOuts.Outputs, out))

func (utxo *UTXOset) Update(block *Block) {
	var value []byte
	db := utxo.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.IsCoinbaseTxn() {
				for _, in := range tx.Vin {
					updatedOuts := OutputsArr{}
					inID := append(utxoPrefix, in.TXID...)
					item, err := txn.Get(inID)
					HandleErr(err)
					err = item.Value(func(val []byte) error {
						value = val
						return nil
					})
					HandleErr(err)

					outs := DeserializeOutputs(value)

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							panic(err)
						}

					} else {
						if err := txn.Set(inID, updatedOuts.SerializeOutputs()); err != nil {
							panic(err)
						}
					}
				}
			}

			newOutputs := OutputsArr{}
			
			newOutputs.Outputs = append(newOutputs.Outputs, tx.Vout...)
		

			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.SerializeOutputs()); err != nil {
				panic(err)
			}
		}

		return nil
	})
	HandleErr(err)
}

func (utxo UTXOset) CountTrxs() int{
	db := utxo.Blockchain.Database
	counter := 0
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		iter := txn.NewIterator(opts)
		defer iter.Close()
		for iter.Seek(utxoPrefix); iter.ValidForPrefix(utxoPrefix); iter.Next(){
			counter++
		} 
		return nil
	})
	HandleErr(err)
	return counter
}


