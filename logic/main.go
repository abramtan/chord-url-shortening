package main

import (
	"fmt"
	"logic/node"
	"time"
)

func main() {
	nodeAr := make([]*node.Node, 0)

	// Initialize nodes
	for i := 0; i < 10; i++ {
		time.Sleep(1000)
		currNode := node.InitNode(&nodeAr)
		go currNode.Maintain() // fix_fingers, stabilise, check_pred
	}

	time.Sleep(time.Second * 2)

	fmt.Print("testing for short and long url storing and generation")

	// testing URL Shortening and Retrieval
	clientNode := node.InitClient()

	longURL := "http://example.com/long4-trial"
	shortURL := string(clientNode.GenerateShortURL(node.LongURL(longURL)))
	finalIP := clientNode.ClientSendStoreURL(longURL, shortURL, nodeAr)
	fmt.Println("Reached Final IP", finalIP)
	time.Sleep(10 * time.Millisecond)

	retrievedEntry, found := clientNode.ClientRetrieveURL(shortURL, nodeAr)

	fmt.Println("retrieve entry", retrievedEntry, "found", found)
	if found {
		fmt.Printf("URL Retrieved: %s -> %s\n", retrievedEntry.ShortURL, retrievedEntry.LongURL)
	} else {
		fmt.Println("URL not found")
	}

	for _, node := range nodeAr {
		fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
	}
}
