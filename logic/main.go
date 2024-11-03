package main

import (
	"fmt"
	"logic/node"
	"math"
	"sort"
	"time"
)

func setFingersStatic(nodeAr *[]*node.Node) {
	sort.Slice(*nodeAr, func(i, j int) bool {
		return (*nodeAr)[i].GetIPAddress().GenerateHash() < (*nodeAr)[j].GetIPAddress().GenerateHash()
	})

	// len(*nodeAr) needs to match the estimated size given in GenerateHash (since thats the size of the ring)
	for i := 0; i < len(*nodeAr); i++ {
		fmt.Printf("I: %d\n", i)
		n := (*nodeAr)[i]
		for j := 0; j < len(*nodeAr); j++ {
			___n := n.GetIPAddress()
			__n := n.GetIPAddress().GenerateHash()
			_n := float64(n.GetIPAddress().GenerateHash())
			_2kminus1 := math.Pow(2, float64(j))
			_2m := math.Pow(2, float64(len(*nodeAr)))
			fmt.Printf("J: %d -- %d -- %d -- %f -- %f -- %f\n", j, ___n, __n, _n, _2kminus1, _2m)
			threshold := math.Mod((float64(n.GetIPAddress().GenerateHash()) + math.Pow(2, float64(j))), math.Pow(2, float64(len(*nodeAr))))

			appended := false
			for k := 0; k < len(*nodeAr); k++ {
				fmt.Printf("K: %d: compare %d with threshold %d\n", k, (*nodeAr)[k].GetIPAddress().GenerateHash(), node.Hash(threshold))
				if (*nodeAr)[k].GetIPAddress().GenerateHash() > node.Hash(threshold) {
					// n.GetFingerTable()[j] = (*nodeAr)[k].GetIPAddress()
					appended = true
					(*(n.GetFingerTable())) = append((*(n.GetFingerTable())), (*nodeAr)[k].GetIPAddress())
					break
				}
			}
			if !appended {
				(*(n.GetFingerTable())) = append((*(n.GetFingerTable())), (*nodeAr)[0].GetIPAddress())
			}
		}
	}
}

// Hardcoding successors
func setSuccessor(nodeAr *[]*node.Node) {
	sort.Slice(*nodeAr, func(i, j int) bool {
		return (*nodeAr)[i].GetIPAddress().GenerateHash() < (*nodeAr)[j].GetIPAddress().GenerateHash()
	})
	for i, x := range *nodeAr {
		next := (i + 1) % 10
		x.SetSuccessor((*nodeAr)[next].GetIPAddress())
	}
}

func main() {
	// size of ring
	// var M float64
	// M = 10

	var nodeAr []*node.Node
	nodeAr = make([]*node.Node, 0)

	// Initialize nodes
	for i := 0; i < 10; i++ {
		time.Sleep(1000)
		node.InitNode(&nodeAr)
	}

	setFingersStatic(&nodeAr)
	setSuccessor(&nodeAr)
	
	for _, x := range nodeAr {
		fmt.Printf("HI THERE %+v, %d\n", x, x.GetIPAddress().GenerateHash())
		go x.Run()
	}

	fmt.Print("testing for short and long url storing and generation")

	// testing URL Shortening and Retrieval
	longURL := node.LongURL("http://example.com/long4-trial")
	shortNode := nodeAr[0]
	shortURL := shortNode.GenerateShortURL(longURL)

	shortNode.StoreURL(shortURL, longURL)
	retrievedURL, found := shortNode.RetrieveURL(shortURL)

	if found {
		fmt.Printf("URL Retrieved: %s -> %s\n", shortURL, retrievedURL)
	} else {
		fmt.Println("URL not found")
	}
	time.Sleep(10 * time.Second)
}
