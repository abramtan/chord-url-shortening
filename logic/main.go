package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"logic/node"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	nodeAr     []*node.Node
	clientNode *node.Node
	menuLog    *log.Logger
	longURLAr  []string
	shortURLAr []string
	// NUMNODES   int
	// numURLs    int
)

func main() {
	go func() {
		node.InfoLog.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.SetOutput(io.Discard)

	nodeAr = make([]*node.Node, 0)

	// testing URL Shortening and Retrieval
	clientNode = node.InitClient()

	log.SetOutput(io.Discard)

	menuLog = log.New(os.Stdout, "MENU: ", 0)

	longURLAr = make([]string, 0)
	shortURLAr = make([]string, 0)

	time.Sleep(1500)
	showmenu(menuLog)

	// Initialisation of the Ring
	time.Sleep(5 * time.Millisecond)
	menuLog.Println("*******************************")
	menuLog.Println("Initialise values for the ring:")
	menuLog.Println("*******************************")

	var NUMNODES int
	var numURLs int

	menuLog.Println("Enter number of nodes in the Chord Ring:")
	fmt.Scanln(&NUMNODES)
	menuLog.Println("Enter number of URLs to store in the Chord Ring:")
	fmt.Scanln(&numURLs)

	menuLog.Println("Initialising Ring...")

	// switch off logs for init
	node.InfoLog.SetOutput(io.Discard)
	initStart := time.Now()
	for i := 0; i < NUMNODES; i++ {
		time.Sleep(1000)
		currNode := node.InitNode(&nodeAr)
		go currNode.Maintain()
		currNode.InitSuccList()
	}
	initEnd := time.Now()
	menuLog.Printf("---------------------------------------------------------------------\n")
	menuLog.Printf("Time taken to init %v nodes: %v\n", NUMNODES, initEnd.Sub(initStart))
	menuLog.Printf("---------------------------------------------------------------------\n")

	file, err := os.Open("URLs.csv")
	if err != nil {
		menuLog.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read()

	for i := 0; i < numURLs; i++ {
		record, err := reader.Read()
		if err != nil {
			break
		}
		longURLAr = append(longURLAr, record[0])
	}

	menuLog.Println("Storing URLS...")

	storeStart := time.Now()
	for _, val := range longURLAr {
		short := string(clientNode.GenerateShortURL(node.LongURL(val)))
		ipAddr := clientNode.ClientSendStoreURL(val, short, nodeAr)
		shortURLAr = append(shortURLAr, short)
		node.InfoLog.Println("Reached Final IP", ipAddr, "for val", val)
	}
	node.InfoLog.SetOutput(os.Stdout)
	storeEnd := time.Now()
	menuLog.Printf("---------------------------------------------------------------------\n")
	menuLog.Printf("Time taken to store URLs: %v\n", storeEnd.Sub(storeStart))
	menuLog.Printf("---------------------------------------------------------------------\n")

	time.Sleep(1 * time.Second)

	// Start the Gin API server in a separate goroutine
	go startAPIServer()

	for {
		time.Sleep(5 * time.Millisecond)
		var input string
		menuLog.Println("***********************************************************************************************")
		menuLog.Println("Enter ADD, DEL, EXPERIMENT, FAULT, FIX, MENU, RETRIEVE, RETRIEVEALL, SHOW, STORE, TESTCACHE:")
		menuLog.Println("***********************************************************************************************")
		fmt.Scanln(&input)

		switch input {
		case "ADD":
			newNode := node.InitNode(&nodeAr)
			menuLog.Println("Adding new node", newNode.GetIPAddress())
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
			longURLAr = append(longURLAr, LONGURL)
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
			var numTimes int
			menuLog.Println("Type 'YES' if you would like to see logs.")
			fmt.Scanln(&SHOWLOGS)
			if SHOWLOGS == "YES" {
				menuLog.Println("Showing Logs...")
			} else {
				node.InfoLog.SetOutput(io.Discard)
			}
			menuLog.Println("Type number of times to run the whole array")
			fmt.Scanln(&numTimes)
			var noCacheTime time.Duration
			var cacheTime time.Duration
			var noCacheCalls int
			var cacheCalls int
			// numTimes = int(numTimes)
			for i := 0; i < numTimes; i++ {
				for _, short := range shortURLAr {
					ncCall, ncTime := retrieveAndMeasure(short, nodeAr, clientNode, "nocache")
					noCacheTime += ncTime
					noCacheCalls += ncCall
					cCall, cTime := retrieveAndMeasure(short, nodeAr, clientNode, "cache")
					cacheTime += cTime
					cacheCalls += cCall
				}
			}

			menuLog.Println("FINAL EXPERIMENT STATISTICS")
			menuLog.Println("No Cache Time:", noCacheTime, "No Cache Calls:", noCacheCalls, "Average of No Cache Time:", noCacheTime/time.Duration(numTimes*len(shortURLAr)))
			menuLog.Println("Cache Time:", cacheTime, "Cache Calls:", cacheCalls, "Average of Cache Time:", cacheTime/time.Duration(numTimes*len(shortURLAr)))

			if !(SHOWLOGS == "YES") {
				node.InfoLog.SetOutput(os.Stdout)
			}
		case "EXPERIMENT":
			var SHOWLOGS string
			var CACHELIMIT string
			var CACHELIMITVALUE int
			var numCalls int

			menuLog.Println("Type 'YES' if you would like to see logs.")
			fmt.Scanln(&SHOWLOGS)
			if SHOWLOGS == "YES" {
				menuLog.Println("Showing Logs...")
			} else {
				menuLog.Println("Discarding Logs...")
				node.InfoLog.SetOutput(io.Discard)
			}

			menuLog.Println("Enter number of random retrievals to make:")
			fmt.Scanln(&numCalls)

			menuLog.Println("Type 'YES' if you would like to set a cache limit.")
			fmt.Scanln(&CACHELIMIT)
			if CACHELIMIT == "YES" {
				node.CacheLimiter = true
			} else if CACHELIMIT == "NO" {
				node.CacheLimiter = false
			} else {
				menuLog.Println("ERROR: not a valid value, defaulting to YES")
			}

			menuLog.Println("Type a number below 100 for Cache Limit.")
			fmt.Scanln(&CACHELIMITVALUE)
			if CACHELIMITVALUE < 100 {
				node.CacheLimit = CACHELIMITVALUE
			}

			menuLog.Println("Running Experiment with", NUMNODES, "nodes and", numURLs, "URLs, with a total of", numCalls, "randomized retrieval calls")
			reader := csv.NewReader(file)
			reader.Read()

			time.Sleep(1 * time.Second)

			var noCacheTime time.Duration
			var cacheTime time.Duration
			var noCacheCalls int
			var cacheCalls int
			for i := 0; i < numCalls; i++ {
				short := shortURLAr[rand.Intn(len(shortURLAr))]

				ncCall, ncTime := retrieveAndMeasure(short, nodeAr, clientNode, "nocache")
				noCacheTime += ncTime
				noCacheCalls += ncCall
				cCall, cTime := retrieveAndMeasure(short, nodeAr, clientNode, "cache")
				cacheTime += cTime
				cacheCalls += cCall
			}

			menuLog.Println("FINAL EXPERIMENT STATISTICS for", NUMNODES, "nodes and", numURLs, "URLs, with a total of", numCalls, "randomized retrieval calls")
			menuLog.Println("No Cache Time:", noCacheTime, "No Cache Calls:", noCacheCalls, "Average of No Cache Time:", noCacheTime/time.Duration(numCalls))
			menuLog.Println("Cache Time:", cacheTime, "Cache Calls:", cacheCalls, "Average of Cache Time:", cacheTime/time.Duration(numCalls))

			if !(SHOWLOGS == "YES") {
				node.InfoLog.SetOutput(os.Stdout)
			}
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
		case "TESTCACHE":
			var RETRIEVEIP string
			var SHORT string
			menuLog.Println("Type IP Address to Retrieve URL From")
			fmt.Scanln(&RETRIEVEIP)
			menuLog.Println("Type in Short URL to retrieve")
			fmt.Scanln(&SHORT)

			// retrieve appropiate node
			var retrNode *node.Node
			for _, no := range nodeAr {
				if no.GetIPAddress() == node.HashableString(RETRIEVEIP) {
					retrNode = no
					break
				}
			}

			if retrNode.GetIPAddress() == node.HashableString(RETRIEVEIP) {
				tempAr := []*node.Node{retrNode}
				// retrieveAndMeasure(SHORT, tempAr, clientNode, "nocache")
				retrieveAndMeasure(SHORT, tempAr, clientNode, "cache")
			} else {
				menuLog.Println("Invalid input...")
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

// TODO: Clean up the rest of the code
// TODO: add Init function so that we can set num nodes and num urls
// TODO: Experiment can be set to clear before adding in

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
	menuLog.Println("Send TESTCACHE to run experiment for average time + hopcount")
	menuLog.Println("Send FAULT to fail a specific node")
	menuLog.Println("Send FIX to revive a specific node")
	menuLog.Println("Send SHOW to current status of all nodes")
	menuLog.Println("Send EXPERIMENT to set specific variables")
	menuLog.Println("Press MENU to see the menu")
	menuLog.Println("****************************************************************")
}

func startAPIServer() {
	r := gin.Default()

	// r.POST("/add", func(c *gin.Context) {
	// 	handleAddNode()
	// 	c.JSON(http.StatusOK, gin.H{"status": "Node added"})
	// })

	// r.POST("/del", func(c *gin.Context) {
	// 	var json struct {
	// 		IP string `json:"ip"`
	// 	}
	// 	if err := c.ShouldBindJSON(&json); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}

	// 	if handleDeleteNodeByIP(json.IP) {
	// 		c.JSON(http.StatusOK, gin.H{"status": "Node deleted"})
	// 	} else {
	// 		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid IP Address"})
	// 	}
	// })

	// r.POST("/fault", func(c *gin.Context) {
	// 	var json struct {
	// 		IP string `json:"ip"`
	// 	}
	// 	if err := c.ShouldBindJSON(&json); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}

	// 	if handleFaultNodeByIP(json.IP) {
	// 		c.JSON(http.StatusOK, gin.H{"status": "Node faulted"})
	// 	} else {
	// 		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid IP Address"})
	// 	}
	// })

	// r.POST("/fix", func(c *gin.Context) {
	// 	var json struct {
	// 		IP string `json:"ip"`
	// 	}
	// 	if err := c.ShouldBindJSON(&json); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}

	// 	if handleFixNodeByIP(json.IP) {
	// 		c.JSON(http.StatusOK, gin.H{"status": "Node fixed"})
	// 	} else {
	// 		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid IP Address"})
	// 	}
	// })

	r.POST("/store", func(c *gin.Context) {
		var json struct {
			LongURL string `json:"long_url"`
		}
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		short_url, short_code, ip, timeTaken := handleStoreURLAPI(json.LongURL)
		c.JSON(http.StatusOK, gin.H{"short_url": short_url, "short_code": short_code, "ip": ip, "time_taken": timeTaken.String()})
	})

	r.GET("/retrieve", func(c *gin.Context) {
		shortURL := c.Query("short_url")
		if !slices.Contains(shortURLAr, shortURL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ShortURL"})
			return
		}

		acquiredURL, calls, found := clientNode.ClientRetrieveURL(shortURL, nodeAr, "nocache")

		if found {
			c.JSON(http.StatusOK, gin.H{
				"long_url":  acquiredURL.LongURL,
				"short_url": acquiredURL.ShortURL,
				"calls":     calls,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"long_url":   "not found",
				"short_code": shortURL,
			})
		}

	})

	// r.GET("/show", func(c *gin.Context) {
	// 	info := handleShowAPI()
	// 	c.JSON(http.StatusOK, info)
	// })

	r.Run(":8080")
}

func handleStoreURLAPI(LONGURL string) (string, string, string, time.Duration) {
	storeStart := time.Now()
	longURLAr = append(longURLAr, LONGURL)
	tempShort := string(clientNode.GenerateShortURL(node.LongURL(LONGURL)))
	shortURLAr = append(shortURLAr, tempShort)
	successIP := clientNode.ClientSendStoreURL(LONGURL, tempShort, nodeAr)
	storeEnd := time.Now()

	base_url := "http://localhost:3000/"
	short_url := base_url + tempShort

	return short_url, tempShort, string(successIP), storeEnd.Sub(storeStart)
}
