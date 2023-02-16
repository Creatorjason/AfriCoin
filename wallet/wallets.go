package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
)

const walletFile = "./tmp/wallets_%s.data"
// tmp/wallets.data

 
type WalletsFile struct{
	Wallets map[string]*Wallet
}

//  

func (wf *WalletsFile) SaveFile(nodeId string) {
	var buf bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeId)
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(wf)
	HandleErr(err)
	err = ioutil.WriteFile(walletFile, buf.Bytes(), 0644)
	HandleErr(err)

}

func (wf *WalletsFile) LoadFile(nodeId string) error{
	var wallet WalletsFile
	walletFile := fmt.Sprintf(walletFile, nodeId)
	if _, err := os.Stat(walletFile); os.IsNotExist(err){
		return err
	}
	content , err := ioutil.ReadFile(walletFile)
	HandleErr(err)
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))
	err = decoder.Decode(&wallet)
	HandleErr(err)
	wf.Wallets = wallet.Wallets
	return nil
}

func CreateWallets(nodeId string) (*WalletsFile , error){
	wallets := WalletsFile{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile(nodeId)
	return &wallets, err
}

func (wf WalletsFile) GetWallet(address string) Wallet{
	return *wf.Wallets[address]
}

func (wf *WalletsFile) GetAllAddress() []string{
	var addresses []string
	for address := range wf.Wallets{
		addresses = append(addresses, address)
	}
	return addresses
}

func (wf *WalletsFile) AddWallet() string{
	newWallet := MakeWallet()
	address := fmt.Sprintf("%s\n", newWallet.Address())
	wf.Wallets[address] = newWallet
	return address
}

