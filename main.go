package main

import (
	"os"
	// "fmt"
	
	"main.go/cli"
	// "main.go/wallet"
	
)



func main(){
	defer os.Exit(0)


	cmd := cli.CommandLine{}
	cmd.Run()

	// wallet := wallet.MakeWallet()
	// fmt.Printf("address: %s",wallet.Address())
}

// 1HjdGSjiyR8nTfrjCM3pkcMVenVjYWnsr5 mined
// 12uN1x32XXt8FneTAu3AvogSg22hZa6GCS