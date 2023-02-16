package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/vrecan/death/v3"
	"main.go/blockchain"
	// "bytes"
)

const(
	protocol = "tcp"
	nVersion = 1
	commandLen = 12
)

var(
	nodeAddr string  // port address of node
	minerAddr string // port address of miner
	KnownNodes = []string{"localhost:3000"} // A list of known nodes addresses
	blocksInTransit = [][]byte{} //No of blocks in transit, usually 500 in the Bitcoin protocol
	memPool = make(map[string]blockchain.Transaction) // Map TXID to tx
)

type Addr struct{
	AddrList []string // List of addresses of nodes connected to the network
}

type Block struct{
	AddrYou string // The address the block is being built from
	Block []byte
}

type GetBlocks struct{
	AddrYou string
	// Get the block from one node and send(copy) to another node
}

type GetData struct{
	AddrYou   string
	Type      string
	ID        []byte
}

type Inventory struct{
	AddrYou   string
	Type      string
	Items     [][]byte
}

type TX struct{
	AddrYou     string
	Transaction []byte
}

type Version struct{
	Version     int
	BestHeight  int
	AddrYou     string
}

func handleErr(err error){
	if err != nil{
		panic(err)
	}
}

func CmdToBytes(cmd string) []byte{
	var bytes [commandLen]byte

	for i, c := range cmd{
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func BytesToCmd(bytes []byte) string{
	var cmd []byte

	for _, b := range bytes{
		if b != 0x0{
			cmd = append(cmd, b)
		}
	}
	return fmt.Sprintf("%s", cmd)
}

func ExtractCmd(request []byte) []byte{
	return request[:commandLen]
}
func GobEncode(data interface{}) []byte{
	buff := new(bytes.Buffer)
	encoder := gob.NewEncoder(buff)
	err := encoder.Encode(data)
	handleErr(err)
	return buff.Bytes()
}

func CloseDB(chain *blockchain.BlockChain){
	close := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt) // Linux, OSx, Windows

	close.WaitForDeathWithFunc(func (){
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}

func HandleConn(conn net.Conn, chain *blockchain.BlockChain){
	req, err := ioutil.ReadAll(conn)  // Reads from the conn until EOF
	defer conn.Close()

	handleErr(err)
	command := BytesToCmd(req[:commandLen])
	fmt.Printf("Received %s command \n", command)
	switch command{
	case "addr":
		HandleAddr(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInventory(req, chain)
	case "tx":
		HandleTx(req, chain)
	case "version":
		HandleVersion(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)


	default:
		fmt.Println("Invalid command")
	}
}

func SendData(addr string, data []byte){
	conn, err := net.Dial(protocol, addr)
	if err != nil{
		fmt.Printf("%s is not available \n", addr)
		var updatedNodes []string

		for  _, node := range KnownNodes{
		if node != addr{
			updatedNodes = append(updatedNodes, node)
			}
		} 
		KnownNodes = updatedNodes
		return
}
	defer conn.Close()
	_, err = io.Copy(conn, bytes.NewReader(data))
	handleErr(err)
} 

func SendAddr(address string){
	nodes := Addr{KnownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddr)
	payload := GobEncode(nodes)
	request := append(CmdToBytes("addr"), payload...)

	SendData(address, request)
}

func SendBlock(address string, block *blockchain.Block){
	data := Block{nodeAddr, block.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("block"), payload...)
	
	SendData(address, request)
}

func SendInventory(addr, kind string, items [][]byte){
	data := Inventory{addr, kind, items}
	payload := GobEncode(data)
	request := append(CmdToBytes("inv"), payload...)

	SendData(addr, request)
}

func SendTx(addr string, transaction *blockchain.Transaction){
	data := TX{addr, transaction.SerializeTx()} // Pilot Police 😎
	payload := GobEncode(data)
	request := append(CmdToBytes("tx"), payload...)
	SendData(addr, request)
}

func SendVersion(addr string, chain *blockchain.BlockChain){
	bestHeight := chain.GetBestHeight()
	data := Version{nVersion, bestHeight, addr}
	payload := GobEncode(data)
	request := append(CmdToBytes("version"), payload...)
	SendData(addr, request)
}

func SendGetBlocks(addr string){
	data := GetBlocks{nodeAddr}
	payload := GobEncode(data)
	request := append(CmdToBytes("getblocks"), payload...)
	SendData(addr, request)
}

func SendGetData(addr, kind string, id []byte){
	data := GetData{nodeAddr, kind, id}
	payload := GobEncode(data)
	request := append(CmdToBytes("getdata"), payload...)
	SendData(addr, request)
}

func HandleAddr(request []byte){
	var buff bytes.Buffer
	var payload Addr

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)
	KnownNodes = append(KnownNodes, payload.AddrList...)
	RequestBlocks() 
}

func RequestBlocks(){
	for _, node := range KnownNodes{
		SendGetBlocks(node )
	}
}

func HandleBlock(request []byte, chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload Block

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)
	blockData := payload.Block
	block := blockchain.Deserialize(blockData)
	fmt.Println("Received a new block")
	chain.AddBlock(block) // CHANGE LATER
	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0{
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrYou, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	}else{
		UTXOSet := blockchain.UTXOset{chain}
		UTXOSet.Reindex()
	}
}

func HandleGetBlocks(request []byte, chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)
	blocks := chain.GetBlocksHashes()
	SendInventory(payload.AddrYou, "blocks", blocks)
}

func HandleGetData(request []byte, chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload GetData

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)

	if payload.Type == "block"{
		block, err := chain.GetBlock([]byte(payload.ID))
		handleErr(err)
		SendBlock(payload.AddrYou, &block)
	}

	if payload.Type == "tx"{
		txId := hex.EncodeToString(payload.ID)
		tx := memPool[txId]

		SendTx(payload.AddrYou, &tx)
	}
}

func HandleVersion(request []byte ,chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)
	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight{
		SendGetBlocks(payload.AddrYou)
	}else if bestHeight > otherHeight{
		SendVersion(payload.AddrYou, chain)
	}

	if !NodeIsKnown(payload.AddrYou){
		KnownNodes = append(KnownNodes, payload.AddrYou)

	}

}

func NodeIsKnown(addr string) bool{
	for _, node := range KnownNodes{
		if node == addr{
			return true
		}
	}
	return false
}

func HandleTx(request []byte, chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload TX

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)

	txData := payload.Transaction
	tx := blockchain.DeserializeTrx(txData)
	memPool[hex.EncodeToString(tx.ID)] =  tx
	
	if nodeAddr == KnownNodes[0]{
		for _, node := range KnownNodes{
			if node != nodeAddr && node != payload.AddrYou{
				SendInventory(node, "tx", [][]byte{tx.ID})
			}
		}
	}else{
		if len(memPool) >= 2 && len(minerAddr) > 0{
			MineTx(chain)
		}
	}
}

func MineTx(chain *blockchain.BlockChain){
	var txs []*blockchain.Transaction
	for id := range memPool{
		tx := memPool[id]
		if chain.VerifyTx(&tx){
			txs = append(txs, &tx)
		}
	}
	if len(txs) == 0{
		fmt.Println("All transactions are invalid")
		return
	}
	coinbaseTx := blockchain.CoinbaseTx(minerAddr, "")
	txs = append(txs, coinbaseTx)

	newBlock := chain.MineBlock(txs)
	UTXOSet := blockchain.UTXOset{chain}
	UTXOSet.Reindex()

	fmt.Println("New Block mined")

	for _, tx := range txs{
		txID := hex.EncodeToString(tx.ID)
		delete(memPool, txID)
	}

	for _, node := range KnownNodes{
		if node != nodeAddr{
			SendInventory(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memPool) > 0{
		MineTx(chain)
	}
}

func HandleInventory(request []byte, chain *blockchain.BlockChain){
	var buff bytes.Buffer
	var payload Inventory

	buff.Write(request[commandLen:])
	decoder := gob.NewDecoder(&buff)
	err := decoder.Decode(&payload)
	handleErr(err)

	if payload.Type == "block"{
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrYou, "block", blockHash)
		
		newInTransit := [][]byte{}
		for _, b := range blocksInTransit{
			if bytes.Compare(b, blockHash) != 0{
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}
	if payload.Type == "tx"{
		txId := payload.Items[0]
		
		if memPool[hex.EncodeToString(txId)].ID == nil{
			SendGetData(payload.AddrYou, "tx", txId)

		}
	}
}

func StartServer(nodeID, minerAddress  string){
	nodeAddr = fmt.Sprintf("localhost:%s", nodeID)
	minerAddress = minerAddr
	ln, err := net.Listen(protocol, nodeAddr)
	handleErr(err)
	defer ln.Close()

	chain := blockchain.ContinueBlockchain(nodeID)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddr != KnownNodes[0]{
		SendVersion(KnownNodes[0], chain)
	}
	for{
		conn, err := ln.Accept()
		handleErr(err)
		go HandleConn(conn, chain)
	}
}
