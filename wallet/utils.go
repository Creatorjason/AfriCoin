package wallet
import (
	// "github.com/mr-tron/base58"
	"github.com/btcsuite/btcutil/base58"
)


func Base58Encode(input []byte) []byte{
	// input := []byte(str)
	encode := base58.Encode(input)

	return []byte(encode)
}

func Base58Decode(input []byte) []byte{
	decode:= base58.Decode(string(input[:]))
	return decode
}

func HandleErr(err error){
	if err != nil{
		panic(err)
	}
}