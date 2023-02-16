package blockchain

import (
	"crypto/sha256"

)

type MerkleTree struct{
	RootNode *MerkleNode
}

type MerkleNode struct{
	Left *MerkleNode
	Right *MerkleNode
	Data []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode{
	node  := MerkleNode{}
	if left == nil && right == nil{
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	}else{
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}
	node.Right = right
	node.Left = left

	return &node
}

func NewMerkleTree(data [][]byte) *MerkleTree{
	var nodes []MerkleNode
	// Checking if the nodes on the merkle tree are even, if not duplicated the last node
	if len(data) % 2 != 0{
		data = append(data, data[len(data) - 1])
	}
	for _, datum := range data{
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}
	for i := 0; i < len(data)/2; i++{
		var level []MerkleNode

		for j := 0; j < len(nodes); j+=2{
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}
		nodes = level
	}
	tree := &MerkleTree{&nodes[0]}
	return tree
}