package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"math"
	"math/big"

)


const DIFFICULTY_BITS = 12

type POW struct{
	Block *Block
	Target *big.Int
}

func ComputeTargetForBlock(b *Block) *POW{
	target := big.NewInt(1)
	target.Lsh(target, uint(256 - DIFFICULTY_BITS))
	return &POW{b, target}
}
func (pow *POW)AssembleBlockDataAndReturnByteRep(nonce int) []byte{
	blockData := bytes.Join([][]byte{
			pow.Block.PrevHash,
			pow.Block.HashTransactions(),
			UtilConvertIntToByteRep(int64(nonce)),
			UtilConvertIntToByteRep(int64(DIFFICULTY_BITS)),
	}, []byte{})

	return blockData
}

func UtilConvertIntToByteRep(num int64) []byte{
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil{
		log.Panic(err)
	}
	return buff.Bytes()
}

func (pow *POW) RunPOW() (int, []byte){
	var intRepOfHash big.Int
	var hash [32]byte

	nonce := 0

	for nonce < math.MaxInt64{
		data := pow.AssembleBlockDataAndReturnByteRep(nonce)
		hash = sha256.Sum256(data)
		
		intRepOfHash.SetBytes(hash[:])
		if intRepOfHash.Cmp(pow.Target) == -1{
			break
		}else{
			nonce++
		}
	}
	return nonce, hash[:]
}

func (pow *POW) ValidatePOW() bool{
	var intRepOfHash big.Int

	data := pow.AssembleBlockDataAndReturnByteRep(pow.Block.Nonce)
	hash := sha256.Sum256(data)

	intRepOfHash.SetBytes(hash[:]) 
	return intRepOfHash.Cmp(pow.Target) == -1
}