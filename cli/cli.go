package cli

import (
	"flag"
	"fmt"
	"main.go/blockchain"
	"main.go/network"
	"main.go/wallet"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("getbalance -address ADDRESS - get balance for an address")
	fmt.Println("createblockchain -address ADDRESS creates a blockchain and sends rewards to address ")
	fmt.Println("printchain - prints the blocks in the chain")
	fmt.Println("send -from FROM -to TO - amount AMOUNT -mine - Send amount of coins")
	fmt.Println("createwallet - Creates a new wallet")
	fmt.Println("listaddresses - Lists the addresses in the wallet file")
	fmt.Println(" reindexutxo - Rebuilds the UTXO set")
	fmt.Println("startnode -miner ADDRESS")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain(nodeId string) {
	chain := blockchain.ContinueBlockchain(nodeId)
	defer chain.Database.Close()
	iter := chain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Block Hash: %x \n", block.Hash)
		fmt.Printf("Block PrevHash: %x \n", block.PrevHash)
		// fmt.Printf("Block Data: %s \n", block.Data)
		new := blockchain.ComputeTargetForBlock(block)
		fmt.Printf("POW %s\n", strconv.FormatBool(new.ValidatePOW()))
		for _, tx := range block.Transactions {
			fmt.Println(tx.StringRep())
		}

		if len(block.PrevHash) == 0 {
			break
		}

	}
}

func (cli *CommandLine) StartNode(nodeId, minerAddr string) {
	if len(minerAddr) > 0 {
		if wallet.ValidateAddress(minerAddr) {
			fmt.Println("Mining in progress, address to receive rewards:", minerAddr)
		} else {
			panic("Invalid address")
		}
	}
	network.StartServer(nodeId, minerAddr)
}

func (cli *CommandLine) createBlockchain(nodeId,address string) {
	if !wallet.ValidateAddress(address) {
		panic("Invalid wallet address")
	}
	chain := blockchain.InitializeBlockchain(address, nodeId)
	chain.Database.Close()
	UTXOSet := blockchain.UTXOset{chain}
	UTXOSet.Reindex()

	fmt.Println("Finished")
}

func (cli *CommandLine) getBalance(nodeId,address string) {
	if !wallet.ValidateAddress(address) {
		panic("Invalid wallet address")
	}
	chain := blockchain.ContinueBlockchain(nodeId)
	UTXOSet := blockchain.UTXOset{chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	UTXOs := UTXOSet.FindUTXO(pubKeyHash)
	// "Coins owned by the wallet address owner on the blockchain"
	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to, nodeID string, amount int, mineNow bool) {
	if !wallet.ValidateAddress(from) {
		panic("Invalid wallet address")
	}
	if !wallet.ValidateAddress(to) {
		panic("Invalid wallet address")
	}
	// fmt.Printf("Send called")
	chain := blockchain.ContinueBlockchain(nodeID)
	UTXOSet := blockchain.UTXOset{chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeID)
	blockchain.HandleErr(err)
	wallet := wallets.GetWallet(from)


	tx := blockchain.NewTransaction(&wallet, to, amount, &UTXOSet)
	if mineNow{
		coinBtx := blockchain.CoinbaseTx(to, "")
		txs := []*blockchain.Transaction{coinBtx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	}else{
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("sent tx")
	}


	
	fmt.Println("Success")
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == ""{
		fmt.Println("NODE_ID env not set")
		runtime.Goexit()

	}

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address of the recipient of genesis block reward")
	sendFrom := sendCmd.String("from", "", "Wallet address of sender")
	sendTo := sendCmd.String("to", "", "Wallet address of receiver")
	sendAmount := sendCmd.Int("amount", 0, "Amount to  send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "start mining!")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)

	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		blockchain.HandleErr(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if startNodeCmd.Parsed(){
		nodeID := os.Getenv("NODE_ID")
		if nodeID == ""{
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeID, *startNodeMiner)
	}
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockchain(nodeID,*createBlockchainAddress)
	}
	if printChainCmd.Parsed() {
		// fmt.Println("Unable to print chain")
		cli.printChain(nodeID)
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send( nodeID ,*sendFrom, *sendTo, *sendAmount, *sendMine)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}
	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}
}
func (cli *CommandLine) listAddresses(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	addresses := wallets.GetAllAddress()

	for _, address := range addresses {
		fmt.Println(address)
	}

}
func (cli *CommandLine) createWallet(nodeId string) {
	wallets, _ := wallet.CreateWallets(nodeId)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeId)
	fmt.Printf("New address generated is :%s\n", address)

}
func (cli *CommandLine) reindexUTXO(nodeId string) {
	chain := blockchain.ContinueBlockchain(nodeId)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOset{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTrxs()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}
