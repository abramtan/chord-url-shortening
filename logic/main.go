package main

import (
	"fmt"
	"io"
	"log"
	"logic/node"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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

	// store an array of long urls
	longUrlAr := []string{"www.hello.com", "www.capstone.com", "www.rubbish.com", "www.trouble.com", "www.trouble.com?query=70", "www.distributedsystems.com", "www.golang.com", "www.crying.com"}
	shortUrlAr := make([]string, 0)

	for _, val := range longUrlAr {
		shortVal := string(clientNode.GenerateShortURL(node.LongURL(val)))
		finalIP := clientNode.ClientSendStoreURL(val, shortVal, nodeAr)
		shortUrlAr = append(shortUrlAr, shortVal)
		log.Println("Reached Final IP", finalIP, "for val", val)

	}

	time.Sleep(5 * time.Second)

	for _, short := range shortUrlAr {
		retrShort, shortFound := clientNode.ClientRetrieveURL(short, nodeAr)

		log.Println("retrieve entry", retrShort, "found", shortFound)
		if found {
			log.Printf("URL Retrieved: %s -> %s\n", string(retrShort.ShortURL), retrShort.LongURL)
		} else {
			log.Println("URL not found")
		}
	}

	time.Sleep(5 * time.Second)

	log.SetOutput(io.Discard)
	for _, node := range nodeAr {
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		// fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
		fmt.Println("IP Address: ", node.GetIPAddress())
		fmt.Println("Fix Finger Count:", node.GetFixFingerCount(), " --- Finger Table:", node.GetFingerTable())
		fmt.Println("Successor:", node.GetSuccessor(), " --- Predecessor:", node.GetPredecessor())
		fmt.Println("Successor List:", node.SuccList)
		fmt.Println("URLMap:", node.UrlMap)
	}

	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	leavenode := nodeAr[len(nodeAr)-2]
	fmt.Printf("%+v -- LEAVING: %+v\n", leavenode, leavenode.GetIPAddress().GenerateHash())

	leavenode.Leave()
	nodeAr = append(nodeAr[:len(nodeAr)-2], nodeAr[len(nodeAr)-1:]...)
	time.Sleep(2000)
	fmt.Printf("NODE %s LEFT\n", leavenode.GetIPAddress())
	for _, node := range nodeAr {
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		// fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
		fmt.Println("IP Address: ", node.GetIPAddress())
		fmt.Println("Fix Finger Count:", node.GetFixFingerCount(), " --- Finger Table:", node.GetFingerTable())
		fmt.Println("Successor:", node.GetSuccessor(), " --- Predecessor:", node.GetPredecessor())
		fmt.Println("Successor List:", node.SuccList)
		fmt.Println("URLMap:", node.UrlMap)
	}

	// force program to wait
	longURLAr := make([]node.LongURL, 0)

	// menuLogger exported
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

	for {
		time.Sleep(5 * time.Millisecond)
		var input string
		fmt.Println("********************************")
		fmt.Println("  Enter ADD, DEL, STORE, RETRIEVE, SHOW, LONGURL, MENU:  ")
		fmt.Println("********************************")
		fmt.Scanln(&input)

		switch input {
		case "ADD":
			newNode := node.InitNode(&nodeAr)
			go newNode.Maintain()  // fix_fingers, stabilise, check_pred
			newNode.InitSuccList() // TODO: should this be here?
		case "DEL":
			fmt.Println("Not Implemented Yet")
		case "STORE":
			fmt.Println("Type Long URL to store:")
			var LONGURL string
			fmt.Scanln(&LONGURL)
			longURLAr = append(longURLAr, node.LongURL(LONGURL))
			tempShort := string(clientNode.GenerateShortURL(node.LongURL(LONGURL)))
			successIP := clientNode.ClientSendStoreURL(LONGURL, tempShort, nodeAr) // selects random Node to send to
			fmt.Println("Reached Final IP", successIP)
		case "RETRIEVE":
			fmt.Println("Type Short URL to retrieve:")
			var SHORTURL string
			fmt.Scanln(&SHORTURL)
			acquiredURL, found := clientNode.ClientRetrieveURL(SHORTURL, nodeAr)

			fmt.Println("retrieve entry", acquiredURL, "found", found)
			if found {
				fmt.Printf("URL Retrieved: %s -> %s\n", retrievedEntry.ShortURL, retrievedEntry.LongURL)
			} else {
				fmt.Println("URL not found")
			}
		case "LONGURL":
			fmt.Println(longURLAr)
		case "SHOW":
			for _, node := range nodeAr {
				fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~")
				// fmt.Printf("%+v -- HASH: %+v\n", node, node.GetIPAddress().GenerateHash())
				fmt.Println("IP Address: ", node.GetIPAddress())
				fmt.Println("Fix Finger Count:", node.GetFixFingerCount(), " --- Finger Table:", node.GetFingerTable())
				fmt.Println("Successor:", node.GetSuccessor(), " --- Predecessor:", node.GetPredecessor())
				fmt.Println("Successor List:", node.SuccList)
				fmt.Println("URLMap:", node.UrlMap)
			}
		case "MENU":
			showmenu()
		default:
			fmt.Println("Invalid input...")
		}
	}
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
