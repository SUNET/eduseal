package model

import "gorm.io/gorm"

// Leaf is the database model of a leaf
type Leaf struct {
	gorm.Model
	Value []byte
}

// Leafs is the database model of a leafs
type Leafs []*Leaf

// Empty returns true if the leafs are empty
func (l Leafs) Empty() bool {
	return len(l) == 0
}

// Array returns the leafs as an byte array of arrays
func (l Leafs) Array() [][]byte {
	var data [][]byte
	for _, v := range l {
		data = append(data, v.Value)
	}
	return data
}
