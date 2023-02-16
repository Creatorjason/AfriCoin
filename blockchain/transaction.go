package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"main.go/wallet"
)

type Transaction struct{
	ID []byte
	Vin []TxInputs
	Vout []TxOutputs
}




// Coinbase transaction
func CoinbaseTx(to, data string) *Transaction{
	if data == ""{
		randData := make([]byte, 24)
		_, err := rand.Read(randData)
		HandleErr(err)
		data = fmt.Sprintf("Message: %x", randData)
	}
	txIn := TxInputs{[]byte{}, -1, nil, []byte(data)}
	txOut := NewTxOutput(50, to)

	tx := &Transaction{nil, []TxInputs{txIn}, []TxOutputs{*txOut}}

	tx.ID = tx.HashTx()
	return tx

}

// func (tx * Transaction) SetID(){
// 	var hash [32]byte
// 	buff := new(bytes.Buffer)
// 	encoder := gob.NewEncoder(buff)
// 	err := encoder.Encode(tx)
// 	HandleErr(err)
// 	hash = sha256.Sum256(buff.Bytes())
// 	tx.ID = hash[:]
// }

func (tx *Transaction) IsCoinbaseTxn() bool{
	return len(tx.Vin) == 1 && len(tx.Vin[0].TXID) == 0 && tx.Vin[0].Vout == -1
}

func NewTransaction(w *wallet.Wallet, to string, amount int, utxo *UTXOset) *Transaction{
	var inputs []TxInputs
	var outputs []TxOutputs

	// wallets, err := wallet.CreateWallets()
	// HandleErr(err)
	// w := wallets.GetWallet(from) 
	pubKeyHash := wallet.PubKeyHash(w.PubKey)

	accumulated , validOutputs := utxo.FindSpendableOutputs(pubKeyHash, amount)
	if accumulated < amount{
		panic("Insufficient funds")
	}
	for txid, outs := range validOutputs{
		txID, err := hex.DecodeString(txid)
		HandleErr(err)

		for _,out := range outs{
			input := TxInputs{txID, out,nil, w.PubKey}
			inputs = append(inputs, input)
		}
	}
	from := fmt.Sprintf("%s", w.Address())
	outputs = append(outputs, *NewTxOutput(amount, to))
	if accumulated > amount{
		outputs = append(outputs, *NewTxOutput(accumulated - amount, from))
	}
	tx := &Transaction{nil, inputs, outputs}
	tx.ID = tx.HashTx()
	utxo.Blockchain.SignTrx(tx, w.PrivKey)
	fmt.Println("New transaction created successfully")
	return tx

}

func (tx Transaction) SerializeTx() []byte{
	buff := new(bytes.Buffer)
	encoder := gob.NewEncoder(buff)
	err := encoder.Encode(tx)
	HandleErr(err)
	return buff.Bytes()
}

func (tx *Transaction) HashTx() []byte{
	// Return the hash of tx copy
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.SerializeTx())
	 return hash[:]
}

func (tx *Transaction) TrimmedTxCopy() Transaction{
	var inputs []TxInputs
	var outputs []TxOutputs

	for _, in := range tx.Vin{
		inputs = append(inputs, TxInputs{in.TXID, in.Vout, nil, nil})
	}
	for _, out := range tx.Vout{
		outputs = append(outputs, TxOutputs{out.Value, out.PubKeyHash})
	}
	txCopy := Transaction{tx.ID, inputs, outputs}
	return txCopy
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) {
	if tx.IsCoinbaseTxn(){
		return
	}
	
	for _, in := range tx.Vin{
		if prevTxs[hex.EncodeToString(in.TXID)].ID == nil{
			panic("Error: Previous transactions not found")
		}
	}

	txCopy := tx.TrimmedTxCopy()
	for inId, in := range txCopy.Vin{
		prevTxs := prevTxs[hex.EncodeToString(in.TXID)]
		txCopy.Vin[inId].Sig = nil
		txCopy.Vin[inId].PubKey = prevTxs.Vout[in.Vout].PubKeyHash
		txCopy.ID = tx.HashTx()
		txCopy.Vin[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		HandleErr(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[inId].Sig = signature
	}
}

func (tx *Transaction) Verify(prevTxs map[string]Transaction) bool{
	if tx.IsCoinbaseTxn(){
		return true
	}
	for _, in := range tx.Vin{
		if prevTxs[hex.EncodeToString(in.TXID)].ID == nil {
			panic("Error: Previous transactions not found")
		}
	}

	txCopy := tx.TrimmedTxCopy()
	curve := elliptic.P256()
	for inId, in := range tx.Vin{
		prevTxs := prevTxs[hex.EncodeToString(in.TXID)]
		txCopy.Vin[inId].Sig = nil
		txCopy.Vin[inId].PubKey = prevTxs.Vout[in.Vout].PubKeyHash
		txCopy.ID = tx.HashTx()
		txCopy.Vin[inId].PubKey = nil

		//  Extract pub key
		keyLen := len(in.PubKey)
		x := big.Int{}
		y := big.Int{}

		x.SetBytes(in.PubKey[:keyLen/2])
		y.SetBytes(in.PubKey[(keyLen / 2):])
		
		

		sigLen := len(in.Sig)
		r := big.Int{}
		s := big.Int{}

		r.SetBytes(in.Sig[:(sigLen / 2)])
		s.SetBytes(in.Sig[(sigLen/2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if !ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s){
			return false
		}
	}
	return true
}


func (tx Transaction) StringRep() string{
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction: %x", tx.ID))
	for idx, input := range tx.Vin{
		lines = append(lines, fmt.Sprintf(" Input: %d", idx))
		lines = append(lines, fmt.Sprintf(" TXID:  %x", input.TXID))
		lines = append(lines, fmt.Sprintf(" Vout: %d", input.Vout))
		lines = append(lines, fmt.Sprintf(" Signature: %x", input.Sig))
		lines = append(lines, fmt.Sprintf(" PubKey: %x", input.PubKey))
	}
	
	for idx, output := range tx.Vout{
		lines = append(lines, fmt.Sprintf(" Output ID: %v", idx))
		lines = append(lines, fmt.Sprintf(" Value: %d", output.Value))
		lines = append(lines, fmt.Sprintf(" ScriptPubKey: %x", output.PubKeyHash ))
	}
	return strings.Join(lines, "\n")
}
// 1PVSzWFTkxwHTvp3SfNgNRXzPFQsVDuVRK
// 19Tyq48r4fSXfozb7wDxJEJUbgC8FMJRw7
// 1MLrydCaFy2bYyyckYtrp2SEdn6uE54HMW


























// func (tx Transaction) SerializeTx() []byte{
// 	buff := new(bytes.Buffer)
// 	encoder := gob.NewEncoder(buff)
// 	err := encoder.Encode(tx)
// 	HandleErr(err)
// 	return buff.Bytes()
// }

// func (tx *Transaction) Hash() []byte{
// 	var hash [32]byte
// 	txCopy := *tx
// 	txCopy.ID = []byte{}
// 	hash  = sha256.Sum256(txCopy.SerializeTx())

// 	return hash[:]
// }

// func (tx *Transaction) SignTx(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction){
// 	if tx.IsCoinbaseTxn(){
// 		return
// 	}
// 	for _, in := range tx.Vin{
// 		if prevTxs[hex.EncodeToString(in.TXID)].ID == nil{
// 			panic("Error: No previous transaction found")
// 		}
// 	}
// 	txCopy := tx.TrimmedTxCopy()

// 	for inID, in := range txCopy.Vin{
// 		prevTxs := prevTxs[hex.EncodeToString(in.TXID)]
// 		// Set signature of inputs
// 		txCopy.Vin[inID].Sig  = nil
// 		//  Set pubkey hash to equal txOutput pubkey hash
// 		txCopy.Vin[inID].PubKey = prevTxs.Vout[in.Vout].PubKeyHash
// 		// Hash all transaction
// 		txCopy.ID = tx.Hash()
// 		// Reset pubkey hash to nil
// 		txCopy.Vin[inID].PubKey = nil
// 		// Sign txId
// 		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
// 		HandleErr(err)
// 		signature := append(r.Bytes(), s.Bytes()...)

// 		// Update signature of input 
// 		tx.Vin[inID].Sig = signature

// 	}
// } 
 
// func (tx *Transaction) TrimmedTxCopy() Transaction{
// 	var inputs []TxInputs
// 	var outputs []TxOutputs

// 	for _, in := range tx.Vin{
// 		inputs = append(inputs, TxInputs{in.TXID, in.Vout, nil, nil}) 
// 	}
// 	for _, out := range tx.Vout{
// 		outputs = append(outputs, TxOutputs{out.Value, out.PubKeyHash})
// 	}
// 	txCopy := Transaction{tx.ID, inputs, outputs}
// 	return txCopy
// }

// func (tx *Transaction) VerifyTx(privKey ecdsa.PrivateKey, prevTxs map[string]Transaction) bool{
// 	if tx.IsCoinbaseTxn(){
// 		return true
// 	}
// 	for _, in := range tx.Vin{
// 		if prevTxs[hex.EncodeToString(in.TXID)].ID == nil{
// 			panic("Error: Previous transaction not found")
// 		}
// 	}
// 	txCopy := tx.TrimmedTxCopy()
// 	curve := elliptic.P256()
// 	for inId, in := range txCopy.Vin{
// 		prevTxs := prevTxs[hex.EncodeToString(in.TXID)]
// 		txCopy.Vin[inId].Sig = nil
// 		txCopy.Vin[inId].PubKey = prevTxs.Vout[in.Vout].PubKeyHash
// 		txCopy.ID = tx.Hash()
// 		txCopy.Vin[inId].PubKey = nil
// 		// Extract pub key and signature to verify message was signed by private key	
// 		r := big.Int{}
// 		s := big.Int{}
// 		sigLen := len(in.Sig)
// 		// Divide the signature in half (/2) to extract each part
// 		r.SetBytes(in.Sig[:(sigLen / 2)])
// 		s.SetBytes(in.Sig[(sigLen / 2):])


// 		x := big.Int{}
// 		y := big.Int{}

// 		keyLen := len(in.PubKey)
// 		x.SetBytes(in.PubKey[:(keyLen / 2)])
// 		y.SetBytes(in.PubKey[(keyLen/2):])
		
// 		// Then using ECDSA, we create a new public key with the point x & y
// 		rawPubKey := ecdsa.PublicKey{curve, &x, &y}



// 	}
// }