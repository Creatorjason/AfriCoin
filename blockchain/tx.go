package blockchain

import (
	"bytes"
	"encoding/gob"

	"main.go/wallet"
)

type TxInputs struct{
	TXID []byte
	Vout int
	Sig []byte
	PubKey []byte
}

type TxOutputs struct{
	Value int 
	PubKeyHash []byte
}
type OutputsArr struct{
	Outputs []TxOutputs
}
func (in *TxInputs) UsesKey(pubKeyHash []byte) bool{
	lockingHash := wallet.PubKeyHash(in.PubKey)
	cmp := bytes.Compare(lockingHash, pubKeyHash) 
	return cmp == 0
}

func (out *TxOutputs) Lock(address []byte) {
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1: len(pubKeyHash) - 4]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutputs) IsLockedWithKey(pubKeyHash []byte) bool{
	cmp := bytes.Compare(out.PubKeyHash, pubKeyHash)
	return cmp == 0
}

func NewTxOutput(value int, address string) *TxOutputs {
	newOut := &TxOutputs{value, nil}
	newOut.Lock([]byte(address))

	return newOut
}

func (outs OutputsArr) SerializeOutputs() []byte{
	buff := new(bytes.Buffer)
	encoder := gob.NewEncoder(buff)
	err := encoder.Encode(outs)
	HandleErr(err)
	return buff.Bytes()
} 

func DeserializeOutputs(data []byte) OutputsArr{
	var outs OutputsArr 

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&outs)
	HandleErr(err)

	return outs
}