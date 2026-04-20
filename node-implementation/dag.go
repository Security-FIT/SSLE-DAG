// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

const hashHexLen = 16

func calculateTxHashMasks(txs []*Transaction) {
	// Calculate tx hash mask based on the txId before txs are added into the mempool
	for i, tx := range txs {
		// Find the correct mask for the transaction
		var txInput = 0

		// Create the mask for the transaction
		for i := 0; i < hashHexLen; i++ {
			if tx.Hash[i:][0] < '8' {
				txInput = txInput << 1 // 0
			} else {
				txInput = (txInput << 1) | 1 // 1
			}
		}
		txs[i].HashMask = txInput
	}
}

func doesTxBelongToIndex(tx Transaction, depth int, index int) bool {
	if depth == 0 {
		return index == 0
	}

	mask := index << (hashHexLen - 2 - 1)
	for i := 0; i < depth; i++ {
		if (mask>>(hashHexLen-i-1))^(tx.HashMask>>(hashHexLen-i-1)) != 0 {
			return false
		}
	}
	return true
}

func getSplitSecondIndex(index int, depth int) int {
	if depth < 0 || depth > dagMaxDepth {
		logError(DAG_LOG, "Depth (%d) out of range <0,%d>", depth, dagMaxDepth)
	}

	if depth == 0 {
		depth = dagMaxDepth
	} else {
		depth--
	}

	secondIndex := index + (DAG_PARALLEL_LEN / (1 << (depth + 1)))

	if secondIndex < 0 || secondIndex >= DAG_PARALLEL_LEN {
		logError(DAG_LOG, "Second index (%d) out of range <0,%d> for depth (%d) and index (%d)", secondIndex, DAG_PARALLEL_LEN, depth, index)
	}

	return secondIndex
}

func getMergeSecondIndex(index int, depth int) int {
	if depth == 0 || index%2 != 0 {
		return -1
	}

	if index == 0 {
		return getSplitSecondIndex(index, depth)
	} else {
		if index%32 == 0 {
			if depth < dagMaxDepth-4 {
				return -1
			} else {
				return getSplitSecondIndex(index, depth)
			}
		} else if index%16 == 0 {
			if depth < dagMaxDepth-3 {
				return -1
			} else {
				return getSplitSecondIndex(index, depth)
			}
		} else if index%8 == 0 {
			if depth < dagMaxDepth-2 {
				return -1
			} else {
				return getSplitSecondIndex(index, depth)
			}
		} else if index%4 == 0 {
			if depth < dagMaxDepth-1 {
				return -1
			} else {
				return getSplitSecondIndex(index, depth)
			}
		} else if index%2 == 0 {
			if depth < dagMaxDepth {
				return -1
			} else {
				return getSplitSecondIndex(index, depth)
			}
		} else {
			return -1
		}
	}
}

func decideAction(index int, currentColumn int) DagActionType {
	// ===== 1st: Check if the block is splittable =====

	// Find Parent block
	blockchainMutex.Lock()
	var parentBlock *Block = nil
	for i := len(blockchain) - 1; i >= 0; i-- {
		if blockchain[i].Col == currentColumn-1 && blockchain[i].Row == index {
			// Found Parent block
			parentBlock = &blockchain[i]
			logDebug(DAG_LOG, "Found Parent block %d\n", blockchain[i].Row)
			break
		}

		if blockchain[i].Col < currentColumn-1 {
			break
		}
	}
	blockchainMutex.Unlock()

	if parentBlock == nil {
		return DagActionType{
			Action: DAG_ACTION_SKIP,
			Parent: nil,
			Second: nil,
		}
	}

	if len(parentBlock.Transactions) >= MIN_TX_TO_SPLIT &&
		parentBlock.Depth < dagMaxDepth {
		return DagActionType{
			Action: DAG_ACTION_SPLIT,
			Parent: parentBlock,
			Second: nil,
		}
	}

	// ===== 2nd: Check if the block is mergeable =====

	// Get possible block to merge with
	secondIndex := getMergeSecondIndex(parentBlock.Row, parentBlock.Depth)
	if parentBlock.Depth > 0 && currentColumn > 0 && secondIndex >= 0 {
		// Check if the second index exists
		var secondBlock *Block = nil
		for i := len(blockchain) - 1; i >= 0; i-- {
			if blockchain[i].Col == currentColumn-1 && blockchain[i].Row == secondIndex {
				// Found Second block
				secondBlock = &blockchain[i]
				logDebug(DAG_LOG, "Found Second block %d\n", blockchain[i].Row)
				break
			}

			if blockchain[i].Col < currentColumn-1 {
				break
			}
		}

		if secondBlock != nil &&
			parentBlock.Depth == secondBlock.Depth &&
			len(parentBlock.Transactions)+len(secondBlock.Transactions) <= MIN_TX_TO_MERGE {
			// Check further splits for parentBlock and secondBlock
			return DagActionType{
				Action: DAG_ACTION_MERGE,
				Parent: parentBlock,
				Second: secondBlock,
			}
		}
	}

	// ===== 3rd: The block is continuable =====
	return DagActionType{
		Action: DAG_ACTION_CONTINUE,
		Parent: parentBlock,
		Second: nil,
	}
}
