package main

import (
	"fmt"
	"io"
	"log"
	"logic/node"
	"net/http"
	_ "net/http/pprof"
	"os"
	"slices"
	"time"
)

func main() {

	go func() {
		node.InfoLog.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.SetOutput(io.Discard)

	nodeAr := make([]*node.Node, 0)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// Initialize nodes
	for i := 0; i < node.NUMNODES; i++ {
		time.Sleep(1000)
		currNode := node.InitNode(&nodeAr)
		go currNode.Maintain()  // fix_fingers, stabilise, check_pred, maintain_succ
		currNode.InitSuccList() // TODO: should this be here?
	}

	time.Sleep(time.Second * 2)

	node.InfoLog.Print("testing for short and long url storing and generation")

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
		node.InfoLog.Println("Reached Final IP", finalIP, "for val", val)
	}
	storeEnd := time.Now()
	node.InfoLog.Printf("---------------------------------------------------------------------\n")
	node.InfoLog.Printf("Time taken to store URLs: %v\n", storeEnd.Sub(storeStart))
	node.InfoLog.Printf("---------------------------------------------------------------------\n")

	time.Sleep(5 * time.Second)

	retrieveStart := time.Now()
	for _, short := range insertShort {
		retrShort, _, shortFound := clientNode.ClientRetrieveURL(short, nodeAr, "cache")

		if shortFound {
			node.InfoLog.Printf("URL Retrieved: %s -> %s\n", string(retrShort.ShortURL), retrShort.LongURL)
		} else {
			node.InfoLog.Println("URL not found")
		}
	}

	retrieveEnd := time.Now()
	node.InfoLog.Printf("---------------------------------------------------------------------\n")
	node.InfoLog.Printf("Time taken to retrieve URLs: %v\n", retrieveEnd.Sub(retrieveStart))
	node.InfoLog.Printf("---------------------------------------------------------------------\n")

	time.Sleep(5 * time.Second)
	node.InfoLog.Println("nodeAr:", nodeAr)

	log.SetOutput(io.Discard)

	menuLog := log.New(os.Stdout, "MENU: ", 0)
	// force program to wait
	longURLAr := make([]node.LongURL, 0)
	shortURLAr := make([]string, 0)
	shortURLAr = append(shortURLAr, insertShort...)

	time.Sleep(1500)
	showmenu(menuLog)

	for {
		time.Sleep(5 * time.Millisecond)
		var input string
		menuLog.Println("*******************************************************************************")
		menuLog.Println("   Enter ADD, DEL, STORE, RETRIEVE, RETRIEVEALL, FAULT, FIX, SHOW, LONGURL, MENU:  	")
		menuLog.Println("*******************************************************************************")
		fmt.Scanln(&input)

		switch input {
		case "ADD":
			menuLog.Println("Adding a random new Node")
			newNode := node.InitNode(&nodeAr)
			go newNode.Maintain()  // fix_fingers, stabilise, check_pred
			newNode.InitSuccList() // TODO: should this be here?
		case "DEL":
			menuLog.Println("Type IP Address of Node to Leave:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				leaveNode := nodeAr[idx]
				menuLog.Println("Faulting Node", leaveNode.GetIPAddress())
				leaveNode.Leave()
			} else {
				menuLog.Println("Invalid IP Address of Node")
			}
		case "FAULT":
			menuLog.Println("Type IP Address of Node to Fault:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				faultyNode := nodeAr[idx]
				menuLog.Println("Faulting Node", faultyNode.GetIPAddress())
				faultyNode.Mu.Lock()
				faultyNode.FailFlag = true
				menuLog.Println(faultyNode)
				faultyNode.Mu.Unlock()
			} else {
				menuLog.Println("Invalid IP Address of Node")
			}
		case "FIX":
			menuLog.Println("Type IP Address of Node to Fix:")
			var IP string
			fmt.Scanln(&IP)
			idx := slices.IndexFunc(nodeAr, func(n *node.Node) bool { return string(n.GetIPAddress()) == IP })
			if idx != -1 {
				faultyNode := nodeAr[idx]
				menuLog.Println("Fixing Node", faultyNode.GetIPAddress())
				faultyNode.Mu.Lock()
				faultyNode.FailFlag = false
				menuLog.Println(faultyNode)
				faultyNode.Mu.Unlock()
			} else {
				menuLog.Println("Invalid IP Address of Node")
			}
		case "STORE":
			menuLog.Println("Type Long URL to store:")
			var LONGURL string
			fmt.Scanln(&LONGURL)

			storeStart := time.Now()
			longURLAr = append(longURLAr, node.LongURL(LONGURL))
			tempShort := string(clientNode.GenerateShortURL(node.LongURL(LONGURL)))
			shortURLAr = append(shortURLAr, tempShort)
			successIP := clientNode.ClientSendStoreURL(LONGURL, tempShort, nodeAr) // selects random Node to send to
			menuLog.Println("Reached Final IP", successIP)
			storeEnd := time.Now()

			menuLog.Printf("Time taken to store URLs: %v\n", storeEnd.Sub(storeStart))
		case "RETRIEVE":
			var SHORTURL string
			menuLog.Println(shortURLAr)
			menuLog.Println("Type Short URL to retrieve:")
			fmt.Scanln(&SHORTURL)
			for !slices.Contains(shortURLAr, SHORTURL) {
				menuLog.Println("Invalid ShortURL. Please try again:")
				fmt.Scanln(&SHORTURL)
			}

			menuLog.Printf("~~~~~~~~~~~~~~~~~\n")
			menuLog.Printf("Retrieving URL: %s\n", SHORTURL)

			// Retrieve and measure time for both "nocache" and "cache" modes
			retrieveAndMeasure(SHORTURL, nodeAr, clientNode, "nocache")
			retrieveAndMeasure(SHORTURL, nodeAr, clientNode, "cache")
		case "RETRIEVEALL":
			var SHOWLOGS string
			menuLog.Println("Type 'YES' if you would like to see logs.")
			fmt.Scanln(&SHOWLOGS)
			if SHOWLOGS == "YES" {
				menuLog.Println("Showing Logs...")
			} else {
				node.InfoLog.SetOutput(io.Discard)
			}
			var noCacheTime time.Duration
			var cacheTime time.Duration
			var noCacheCalls int
			var cacheCalls int
			numTimes := 20
			for _, short := range shortURLAr {
				for i := 0; i < numTimes; i++ {
					call, time := retrieveAndMeasure(short, nodeAr, clientNode, "nocache")
					noCacheTime += time
					noCacheCalls += call
					call, time = retrieveAndMeasure(short, nodeAr, clientNode, "cache")
					cacheTime += time
					cacheCalls += call
				}
			}

			menuLog.Println("FINAL EXPERIMENT STATISTICS")
			menuLog.Println("No Cache Time:", noCacheTime, "No Cache Calls:", noCacheCalls, "Average of No Cache Time:", noCacheTime/time.Duration(numTimes*len(shortURLAr)))
			menuLog.Println("Cache Time:", cacheTime, "Cache Calls:", noCacheCalls, "Average of Cache Time:", cacheTime/time.Duration(numTimes*len(shortURLAr)))

			if !(SHOWLOGS == "YES") {
				node.InfoLog.SetOutput(os.Stdout)
			}
		case "LONGURL":
			menuLog.Println(longURLAr)
		case "SHOW":
			for _, printNode := range nodeAr {
				printNode.Mu.Lock()
				menuLog.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				menuLog.Println("IP Address: ", printNode.GetIPAddress(), "-- HASH:", printNode.GetIPAddress().GenerateHash())
				menuLog.Println("FFCount:", printNode.GetFixFingerCount(), " -- Finger Table:", printNode.GetFingerTable())
				menuLog.Println("Successor:", printNode.GetSuccessor(), " -- Predecessor:", printNode.GetPredecessor())
				menuLog.Println("Successor List:", printNode.SuccList)
				menuLog.Println("URLMap:")
				for hashString, mapVal := range printNode.UrlMap.UrlMap {
					menuLog.Println("   for node:", hashString, "-- HASH:", hashString.GenerateHash())
					for short, long := range mapVal {
						menuLog.Println("       for short, long:", short, long, "-- SHORT HASH:", node.HashableString(short).GenerateHash())
					}
				}
				printNode.Mu.Unlock()
			}
		case "MENU":
			showmenu(menuLog)
		default:
			menuLog.Println("Invalid input...")
		}
	}
}

func retrieveAndMeasure(shortURL string, nodeAr []*node.Node, clientNode *node.Node, retrievalMode string) (int, time.Duration) {
	node.InfoLog.Println("============================================================")
	node.InfoLog.Println("Retrieving URL using mode:", retrievalMode)
	retrieveStart := time.Now()

	acquiredURL, calls, found := clientNode.ClientRetrieveURL(shortURL, nodeAr, retrievalMode)
	retrieveEnd := time.Now()

	if found {
		node.InfoLog.Printf("URL Retrieved: %s -> %s\n", acquiredURL.ShortURL, acquiredURL.LongURL)
	} else {
		node.InfoLog.Println("URL not found")
	}

	timeTaken := retrieveEnd.Sub(retrieveStart)
	node.InfoLog.Printf("Time taken to retrieve URL using %s: %v\n", retrievalMode, timeTaken)
	return calls, timeTaken
}

/* Show a list of options to choose from.*/
func showmenu(menuLog *log.Logger) {
	// Enter ADD, DEL, STORE, RETRIEVE, RETRIEVEALL, FAULT, FIX, SHOW, LONGURL, MENU:
	menuLog.Println("****************************************************************")
	menuLog.Println("\t\tMENU")
	menuLog.Println("Send ADD to add node")
	menuLog.Println("Send DEL to delete a random node")
	menuLog.Println("Send STORE to add a new tinyurl")
	menuLog.Println("Send RETRIEVE to get a long url")
	menuLog.Println("Send RETRIEVEALL to run experiment for average time + hopcount")
	menuLog.Println("Send FAULT to fail a specific node")
	menuLog.Println("Send FIX to revive a specific node")
	menuLog.Println("Send SHOW to current status of all nodes")
	menuLog.Println("Send LONGURL to get a list of current long urls")
	menuLog.Println("Press MENU to see the menu")
	menuLog.Println("****************************************************************")
}
