package main

import (
	"logic/node"
	"time"
)

func main() {
	// size of ring
	// var M float64
	// M = 10

	var nodeAr []*node.Node
	nodeAr = make([]*node.Node, 0)

	_, nodeAr = node.InitNode(nodeAr)

	// Keep the parent thread alive
	for {
		time.Sleep(1000)
		node.InitNode(nodeAr)
	}

}
