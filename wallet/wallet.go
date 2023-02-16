package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	// "fmt"
	"golang.org/x/crypto/ripemd160"
	"bytes"
)

const (
	CheckSumLength = 4
	// Hex rep of zero 0x00
	version = byte(0x00)
)

type Wallet struct {
	PrivKey ecdsa.PrivateKey
	PubKey  []byte
}


func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	HandleErr(err)
	pubKey := append(privateKey.X.Bytes(), privateKey.Y.Bytes()...)
	return *privateKey, pubKey
}

func MakeWallet() *Wallet {
	sk, pk := NewKeyPair()
	// JBOK - Just a Bunch of Keys
	wallet := &Wallet{sk, pk}
	return wallet
}

func PubKeyHash(pubkey []byte) []byte {
	hash256 := sha256.Sum256(pubkey)
	hash160 := ripemd160.New()
	_, err := hash160.Write(hash256[:])
	HandleErr(err)
	publicRipMD := hash160.Sum(nil)
	return publicRipMD
}

func CheckSum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	// checksum length slice
	return secondHash[:CheckSumLength]
}

func (wallet Wallet) Address() []byte {
	pubHash := PubKeyHash(wallet.PubKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := CheckSum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	//  stringed := string(fullHash)

	address := Base58Encode(fullHash)
	// fmt.Printf("Full hash: %x\n", fullHash)
	// fmt.Printf("Public Key: %x\n", wallet.PubKey)
	// fmt.Printf("Public key hash: %x\n", pubHash)
	// fmt.Printf("address: %s\n" , address)
	return address
}

func  ValidateAddress(address string) bool{
	pubKeyHash := Base58Decode([]byte(address))
	diff := len(pubKeyHash) - CheckSumLength
	version := pubKeyHash[0]
	actualChecksum := pubKeyHash[diff:]
	pubKeyHash = pubKeyHash[1:diff]
	targetChecksum := CheckSum(append([]byte{version}, pubKeyHash...))
	cmp := bytes.Compare(actualChecksum, targetChecksum)
	return cmp == 0
}