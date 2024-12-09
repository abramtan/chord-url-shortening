package main

import (
	"fmt"
	"io"
	"log"
	"logic/node"

	// "math/rand/v2"
	"net/http"
	_ "net/http/pprof"
	"slices"
	"time"
)

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.SetOutput(io.Discard)

	nodeAr := make([]*node.Node, 0)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// Initialize nodes
	for i := 0; i < 10; i++ {
		time.Sleep(1000)
		currNode := node.InitNode(&nodeAr)
		go currNode.Maintain()  // fix_fingers, stabilise, check_pred, maintain_succ
		currNode.InitSuccList() // TODO: should this be here?
	}

	time.Sleep(time.Second * 2)

	log.Print("testing for short and long url storing and generation")

	// testing URL Shortening and Retrieval
	clientNode := node.InitClient()

	// store an array of long urls
	insertLong := []string{"http://example.com/long4-trial", "www.hello.com", "www.capstone.com", "www.rubbish.com", "www.trouble.com", "www.trouble.com?query=70", "www.distributedsystems.com", "www.golang.com", "www.crying.com"}
	insertShort := make([]string, 0)

	storeStart := time.Now()
	for _, val := range insertLong {
		shortVal := string(clientNode.GenerateShortURL(node.LongURL(val)))
		finalIP := clientNode.ClientSendStoreURL(val, shortVal, nodeAr)
		insertShort = append(insertShort, shortVal)
		log.Println("Reached Final IP", finalIP, "for val", val)

	}
	storeEnd := time.Now()
	fmt.Printf("Time taken to store URLs: %v\n", storeEnd.Sub(storeStart))

	time.Sleep(5 * time.Second)

	retrieveStart := time.Now()
	for _, short := range insertShort {
		retrShort, shortFound := clientNode.ClientRetrieveURL(short, nodeAr, "cache")

		fmt.Println("retrieve entry", retrShort, "found", shortFound)
		if shortFound {
			fmt.Printf("URL Retrieved: %s -> %s\n", string(retrShort.ShortURL), retrShort.LongURL)
		} else {
			fmt.Println("URL not found")
		}
	}

	retrieveEnd := time.Now()
	fmt.Printf("Time taken to retrieve URLs: %v\n", retrieveEnd.Sub(retrieveStart))

	time.Sleep(5 * time.Second)
	fmt.Println("nodeAr:", nodeAr)

	log.SetOutput(io.Discard)
	for _, node := range nodeAr {
		node.Mu.Lock()
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		// fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
		fmt.Println("IP Address: ", node.GetIPAddress())
		fmt.Println("Fix Finger Count:", node.GetFixFingerCount(), " --- Finger Table:", node.GetFingerTable())
		fmt.Println("Successor:", node.GetSuccessor(), " --- Predecessor:", node.GetPredecessor())
		fmt.Println("Successor List:", node.SuccList)
		fmt.Println("URLMap:", node.UrlMap)
		node.Mu.Unlock()
	}

	// force program to wait
	longURLAr := make([]node.LongURL, 0)
	shortURLAr := make([]string, 0)
	shortURLAr = append(shortURLAr, insertShort...)

	time.Sleep(1500)
	showmenu()
	time.Sleep(1500)
	showmenu()

	for {
		time.Sleep(5 * time.Millisecond)
		var input string
		fmt.Println("***************************************************************************")
		fmt.Println("   Enter ADD, DEL, STORE, RETRIEVE, FAULT, FIX, SHOW, LONGURL, MENU:  	")
		fmt.Println("***************************************************************************")
		fmt.Scanln(&input)

		switch input {
		case "ADD":
			fmt.Println("Adding a random new Node")
			newNode := node.InitNode(&nodeAr)
			go newNode.Maintain()  // fix_fingers, stabilise, check_pred
			newNode.InitSuccList() // TODO: should this be here?
		case "DEL":
			fmt.Println("Type IP Address of Node to Leave:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				leaveNode := nodeAr[idx]
				fmt.Println("Faulting Node", leaveNode.GetIPAddress())
				leaveNode.Leave()
				nodeAr = append(nodeAr[:idx], nodeAr[idx+1:]...)
			} else {
				fmt.Println("Invalid IP Address of Node")
			}
		case "FAULT":
			fmt.Println("Type IP Address of Node to Fault:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				faultyNode := nodeAr[idx]
				fmt.Println("Faulting Node", faultyNode.GetIPAddress())
				faultyNode.Mu.Lock()
				faultyNode.FailFlag = true
				fmt.Println(faultyNode)
				faultyNode.Mu.Unlock()
			} else {
				fmt.Println("Invalid IP Address of Node")
			}
		case "FIX":
			fmt.Println("Type IP Address of Node to Fix:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				faultyNode := nodeAr[idx]
				fmt.Println("Fixing Node", faultyNode.GetIPAddress())
				faultyNode.Mu.Lock()
				faultyNode.FailFlag = false
				fmt.Println(faultyNode)
				faultyNode.Mu.Unlock()
			} else {
				fmt.Println("Invalid IP Address of Node")
			}
		case "STORE":
			fmt.Println("Type Long URL to store:")
			var LONGURL string
			fmt.Scanln(&LONGURL)
			storeStart := time.Now()
			longURLAr = append(longURLAr, node.LongURL(LONGURL))
			tempShort := string(clientNode.GenerateShortURL(node.LongURL(LONGURL)))
			shortURLAr = append(shortURLAr, tempShort)
			successIP := clientNode.ClientSendStoreURL(LONGURL, tempShort, nodeAr) // selects random Node to send to
			fmt.Println("Reached Final IP", successIP)
			storeEnd := time.Now()
			fmt.Printf("Time taken to store URLs: %v\n", storeEnd.Sub(storeStart))
		case "RETRIEVE":
			var SHORTURL string
			fmt.Println(shortURLAr)
			fmt.Println("Type Short URL to retrieve:")
			fmt.Scanln(&SHORTURL)
			for !slices.Contains(shortURLAr, SHORTURL) {
				fmt.Println("Invalid ShortURL. Please try again:")
				fmt.Scanln(&SHORTURL)
			}

			// Retrieve and measure time for both "nocache" and "cache" modes
			retrieveAndMeasure(SHORTURL, nodeAr, clientNode, "nocache")
			retrieveAndMeasure(SHORTURL, nodeAr, clientNode, "cache")

		case "LONGURL":
			fmt.Println(longURLAr)
		case "SHOW":
			for _, printNode := range nodeAr {
				printNode.Mu.Lock()
				fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				// fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
				fmt.Println("IP Address: ", printNode.GetIPAddress())
				fmt.Println("Fix Finger Count:", printNode.GetFixFingerCount(), " --- Finger Table:", printNode.GetFingerTable())
				fmt.Println("Successor:", printNode.GetSuccessor(), " --- Predecessor:", printNode.GetPredecessor())
				fmt.Println("Successor List:", printNode.SuccList)
				// fmt.Println("URLMap:", node.UrlMap)
				fmt.Println("URLMap:")
				for hashString, mapVal := range printNode.UrlMap.UrlMap {
					fmt.Println("   for node:", hashString, "-- HASH:", hashString.GenerateHash())
					for short, long := range mapVal {
						fmt.Println("       for short, long:", short, long, "-- SHORT HASH:", node.HashableString(short).GenerateHash())
					}
				}
				printNode.Mu.Unlock()
			}
		case "MENU":
			showmenu()
		default:
			fmt.Println("Invalid input...")
		}
	}
}

func retrieveAndMeasure(shortURL string, nodeAr []*node.Node, clientNode *node.Node, retrievalMode string) {
	fmt.Println("Retrieving URL using mode:", retrievalMode)
	retrieveStart := time.Now()

	acquiredURL, found := clientNode.ClientRetrieveURL(shortURL, nodeAr, retrievalMode)
	retrieveEnd := time.Now()

	fmt.Println("retrieve entry", acquiredURL, "found", found)
	if found {
		fmt.Printf("URL Retrieved: %s -> %s\n", acquiredURL.ShortURL, acquiredURL.LongURL)
	} else {
		fmt.Println("URL not found")
	}

	fmt.Printf("Time taken to retrieve URL using %s: %v\n", retrievalMode, retrieveEnd.Sub(retrieveStart))
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
