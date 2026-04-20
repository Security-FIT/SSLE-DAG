// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"crypto/subtle"
	"encoding/binary"
)

func uInt32ToBytes(i uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, i)
	return buf
}

func int64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func findMajorElementInt64(list []int64) (int64, bool) {
	for _, element := range list {
		count := 0
		for _, element2 := range list {
			if element == element2 {
				count++
			}
		}
		if count > len(list)/2 {
			// Return the majority element
			return element, true
		}
	}

	// No majority element found
	return 0, false
}

func findMajorElementString(list []string) (string, bool) {
	for _, element := range list {
		count := 0
		for _, element2 := range list {
			if element == element2 {
				count++
			}
		}
		if count > len(list)/2 {
			// Return the majority element
			return element, true
		}
	}

	// No majority element found
	return "", false
}

func sliceContainsInt(slice []int, element int) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}

func blockSliceContainsRow(blockSlice []Block, row int) bool {
	for _, block := range blockSlice {
		if block.Row == row {
			return true
		}
	}
	return false
}

/*
	Following function compressPoint is taken and modified from:

github.com/consensys/gnark-crypto for BLS-12-381 twisted edwards curve

It is used to compress the point on curve from 64 bytes to 32 bytes according to RFC 8032, section 3.1
*/
const (
	sizePointCompressed = 32
	mCompressedPositive = 0x00
)

func compressPoint(pY []byte, mask uint) [32]byte {
	var res [sizePointCompressed]byte

	y := pY

	// p.Y must be in little endian
	y[0] |= byte(mask) // msb of y
	for i, j := 0, sizePointCompressed-1; i < j; i, j = i+1, j-1 {
		y[i], y[j] = y[j], y[i]
	}
	subtle.ConstantTimeCopy(1, res[:], y[:])
	return res
}

func logBase2(x int) int {
	if x <= 0 {
		panic("logBase2: x must be greater than 0")
	}
	result := 0
	for x > 1 {
		x >>= 1
		result++
	}
	return result
}

func splitIntoPowersOfTwo(x int, maxPower int) []int {
	var result []int

	if x == 0 {
		for i := 1; i <= maxPower; i <<= 1 {
			result = append(result, i)
		}
		return result
	}

	for x > 0 {
		highestPower := 1
		for highestPower*2 <= x {
			highestPower *= 2
		}
		result = append(result, highestPower)
		x -= highestPower
	}

	return result
}
