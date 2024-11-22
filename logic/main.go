package main

import (
	// "fmt"
	"fmt"
	"io"
	"log"
	"logic/node"
	"time"
)

func main() {
	nodeAr := make([]*node.Node, 0)

	// Initialize nodes
	for i := 0; i < 10; i++ {
		time.Sleep(1000)
		currNode := node.InitNode(&nodeAr)
		go currNode.Maintain()  // fix_fingers, stabilise, check_pred
		currNode.InitSuccList() // TODO: should this be here?
	}

	time.Sleep(time.Second * 2)

	log.Print("testing for short and long url storing and generation")

	// testing URL Shortening and Retrieval
	clientNode := node.InitClient()

	// TODO : Convert this to a terminal string they can type into?

	longURL := "http://example.com/long4-trial"
	shortURL := string(clientNode.GenerateShortURL(node.LongURL(longURL)))
	finalIP := clientNode.ClientSendStoreURL(longURL, shortURL, nodeAr)
	log.Println("Reached Final IP", finalIP)

	time.Sleep(1000)
	currNode := node.InitNode(&nodeAr)
	go currNode.Maintain()  // fix_fingers, stabilise, check_pred
	currNode.InitSuccList() // TODO: should this be here?

	time.Sleep(1 * time.Second)

	retrievedEntry, found := clientNode.ClientRetrieveURL(shortURL, nodeAr)

	log.Println("retrieve entry", retrievedEntry, "found", found)
	if found {
		log.Printf("URL Retrieved: %s -> %s\n", string(retrievedEntry.ShortURL), retrievedEntry.LongURL)
	} else {
		log.Println("URL not found")
	}

	time.Sleep(5 * time.Second)

	log.SetOutput(io.Discard)
	for _, node := range nodeAr {
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
	}

	// // force program to wait
	// longURLAr := make([]node.LongURL, 0)

	// // menuLogger exported
	// var menuLogger *log.Logger

	// absPath, err := filepath.Abs("../logic/log")
	// generalLog, err := os.OpenFile(absPath+"/menu-log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	fmt.Println("Error opening file:", err)
	// 	os.Exit(1)
	// }

	// menuLogger = log.New(generalLog, "Menu Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)
	// log.SetOutput(log.Writer())
	// time.Sleep(1500)
	// showmenu()

	// for {
	// 	time.Sleep(5000)
	// 	var input string
	// 	fmt.Println("********************************")
	// 	fmt.Println("  Enter ADD, DEL, STORE, MENU:  ")
	// 	fmt.Println("********************************")
	// 	fmt.Scanln(&input)

	// 	switch input {
	// 	case "ADD":
	// 		newNode := node.InitNode(&nodeAr)
	// 		go newNode.Maintain()  // fix_fingers, stabilise, check_pred
	// 		newNode.InitSuccList() // TODO: should this be here?
	// 	case "DEL":
	// 		fmt.Println("Not Implemented Yet")
	// 	case "STORE":
	// 		fmt.Println("Type Long URL to store:")
	// 		var LONGURL string
	// 		fmt.Scanln(&LONGURL)
	// 		longURLAr = append(longURLAr, node.LongURL(LONGURL))
	// 		tempShort := string(clientNode.GenerateShortURL(node.LongURL(LONGURL)))
	// 		successIP := clientNode.ClientSendStoreURL(LONGURL, tempShort, nodeAr) // selects random Node to send to
	// 		fmt.Println("Reached Final IP", successIP)
	// 	case "RETRIEVE":
	// 		fmt.Println("Type Short URL to retrieve:")
	// 		var SHORTURL string
	// 		fmt.Scanln(&SHORTURL)
	// 		acquiredURL, found := clientNode.ClientRetrieveURL(SHORTURL, nodeAr)

	// 		fmt.Println("retrieve entry", acquiredURL, "found", found)
	// 		if found {
	// 			fmt.Printf("URL Retrieved: %s -> %s\n", retrievedEntry.ShortURL, retrievedEntry.LongURL)
	// 		} else {
	// 			fmt.Println("URL not found")
	// 		}
	// 	case "LONGURL":
	// 		fmt.Println(longURLAr)
	// 	case "MENU":
	// 		showmenu()
	// 	default:
	// 		fmt.Println("Invalid input...")
	// 	}
	// }
}

/* Show a list of options to choose from.*/
func showmenu() {
	fmt.Println("********************************")
	fmt.Println("\t\tMENU")
	fmt.Println("Send ADD to add node")
	fmt.Println("Send DEL to delete a random node")
	fmt.Println("Send STORE to add a new tinyurl")
	fmt.Println("Send RETRIEVE to get a long url")
	fmt.Println("Send LONGURL to get a list of current long urls")
	fmt.Println("Press MENU to see the menu")
	fmt.Println("********************************")
}
