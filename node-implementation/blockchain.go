// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"encoding/hex"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	"math/big"
	"sync"
	"time"
)

var blockchain []Block
var blockchainLength int
var mempool map[string]*Transaction
var slot int64
var absRound int64

var blockchainMutex sync.Mutex
var mempoolMutex sync.Mutex

var maskConstant *big.Int
var successfulMaskConversion bool

func processGenesisBlock(genesisBlock GenesisBlock) Block {
	// Compute and add missing transaction hashes information
	var txHashes [][]byte
	for i := 0; i < len(genesisBlock.Transactions); i++ {
		txHash := computeTransactionHash(genesisBlock.Transactions[i])
		txHashes = append(txHashes, txHash)
		genesisBlock.Transactions[i].Hash = hex.EncodeToString(txHash)
	}

	merkleRoot := computeMerkleRoot(txHashes)

	block := Block{
		Hash:                    "",
		Number:                  genesisBlock.Number,
		CreatedAt:               genesisBlock.CreatedAt,
		Transactions:            genesisBlock.Transactions,
		MerkleRoot:              hex.EncodeToString(merkleRoot),
		Author:                  genesisBlock.Author,
		PreviousBlock:           nil,
		PreviousBlockHash:       "",
		PreviousSecondBlock:     nil,
		PreviousSecondBlockHash: "",
		Depth:                   0,
		Row:                     0,
		Col:                     0,
		CommitmentHash:          "",
		CommitmentSecret:        "",
		CommitmentSig:           "",
	}

	// Add missing information
	blockHash := hex.EncodeToString(computeBlockHash(block, merkleRoot))
	block.Hash = blockHash

	// Append to blockchain
	blockchainMutex.Lock()
	blockchain = append(blockchain, block)
	blockchainLength++
	blockchainMutex.Unlock()

	logInfo(BCHAIN_LOG, "Genesis block added to blockchain: (%s)", block.Hash[:8])

	return block
}

func processBlock(blockTransfer BlockTransfer, commitmentSharedInfo CommitmentSharedInfo) {
	block := Block{
		Hash:                    blockTransfer.Hash,
		Number:                  blockTransfer.Number,
		CreatedAt:               blockTransfer.CreatedAt,
		Transactions:            blockTransfer.Transactions,
		MerkleRoot:              blockTransfer.MerkleRoot,
		Author:                  blockTransfer.Author,
		PreviousBlock:           nil,
		PreviousBlockHash:       blockTransfer.PreviousBlockHash,
		PreviousSecondBlock:     nil,
		PreviousSecondBlockHash: blockTransfer.PreviousSecondBlockHash,
		Depth:                   blockTransfer.Depth,
		Row:                     blockTransfer.Row,
		Col:                     blockTransfer.Col,
		CommitmentHash:          blockTransfer.CommitmentHash,
		CommitmentSecret:        blockTransfer.CommitmentSecret,
		CommitmentSig:           blockTransfer.CommitmentSig,
	}

	var txHashes [][]byte
	for _, tx := range block.Transactions {
		txHash, err := hex.DecodeString(tx.Hash)
		if err != nil {
			logFatal(BCHAIN_LOG, "Error decoding transaction hash: %s", err)
		}
		txHashes = append(txHashes, txHash)
	}

	merkleRoot := computeMerkleRoot(txHashes)

	// Validate commitment hash
	if block.CommitmentHash != commitmentSharedInfo.Hash {
		logError(BCHAIN_LOG, "Commitment hash (%s) does not match the excepted commitment hash (%s)", block.CommitmentHash, commitmentSharedInfo.Hash)
		return
	}

	// ** Validate block author **
	secretBytes, err := hex.DecodeString(block.CommitmentSecret)
	if err != nil {
		logError(BCHAIN_LOG, "Error decoding secret value in commitment: %s", err)
		return
	}

	authorPublicKeyBytes, err := hex.DecodeString(block.Author)
	if err != nil {
		logError(BCHAIN_LOG, "Error decoding block author public key: %s", err)
		return
	}

	var pubKey eddsa.PublicKey
	_, err = pubKey.SetBytes(authorPublicKeyBytes)
	if err != nil {
		logError(BCHAIN_LOG, "Error setting public key: %s", err)
		return
	}

	// Calculate message sum
	messageBytes := []byte(commitmentSharedInfo.Public.Message)

	hasherSecret := hash.MIMC_BLS12_381.New()
	hasherSecret.Reset()
	hasherSecret.Write(messageBytes)
	hasherSecret.Write(secretBytes)
	messageSummed := hasherSecret.Sum(nil)

	// Create binary secret signature
	messageSigRyBytes, err := hex.DecodeString(commitmentSharedInfo.Public.MessageSigRY)
	if err != nil {
		logError(BCHAIN_LOG, "Error decoding Ry in message signature: %s", err)
		return
	}
	messageSigSBytes, err := hex.DecodeString(commitmentSharedInfo.Public.MessageSigS)
	if err != nil {
		logError(BCHAIN_LOG, "Error decoding S in message signature: %s", err)
		return
	}

	// Compress signature point from 64 bytes to 32 bytes using RFC 8032, section 3.1
	compressedMessageSignaturePointPositive := compressPoint(messageSigRyBytes, mCompressedPositive)

	positivePartMsg := new(big.Int).SetBytes(compressedMessageSignaturePointPositive[:])
	negativePartMsg := new(big.Int).Add(positivePartMsg, maskConstant)

	messageSignatureBytesPositive := append(compressedMessageSignaturePointPositive[:], messageSigSBytes...)
	messageSignatureBytesNegative := append(negativePartMsg.Bytes(), messageSigSBytes...)

	// Validate Message signature
	messageVerify, err := pubKey.Verify(messageSignatureBytesPositive, messageSummed, hash.Hash.New(hash.MIMC_BLS12_381))
	if err != nil {
		logError(BCHAIN_LOG, "Error verifying message signature: %s", err)
	}
	if !messageVerify {
		messageVerify, err = pubKey.Verify(messageSignatureBytesNegative, messageSummed, hash.Hash.New(hash.MIMC_BLS12_381))

		if err != nil {
			logError(BCHAIN_LOG, "Error verifying message signature: %s", err)
		}

		if !messageVerify {
			logError(BCHAIN_LOG, "Error verifying message signature: %v", messageVerify)
		}
	}

	// Validate block hash
	if block.Hash != hex.EncodeToString(computeBlockHash(block, merkleRoot)) {
		logError(BCHAIN_LOG, "Block hash is invalid: (%s) createdBy (%s)", block.Hash, block.Author)
		return
	}

	// Add previous block references
	for i := len(blockchain) - 1; i >= 0; i-- {
		if blockchain[i].Hash == block.PreviousBlockHash {
			block.PreviousBlock = &blockchain[i]
		}
	}
	if block.PreviousBlock == nil {
		logError(BCHAIN_LOG, "Cannot find reference for previous block")
	}

	// Find and add previous Second block reference if present
	if block.PreviousSecondBlockHash != "" {
		for i := len(blockchain) - 1; i >= 0; i-- {
			if blockchain[i].Hash == block.PreviousSecondBlockHash {
				block.PreviousSecondBlock = &blockchain[i]
			}
		}
		if block.PreviousSecondBlock == nil {
			logError(BCHAIN_LOG, "Cannot find reference for previous Second block")
		}
	}

	// Append to blockchain
	blockchainMutex.Lock()
	blockchain = append(blockchain, block)
	blockchainLength++
	blockchainMutex.Unlock()

	// Remove transactions from the mempool
	mempoolMutex.Lock()
	for _, tx := range txHashes {
		delete(mempool, hex.EncodeToString(tx))
	}
	mempoolMutex.Unlock()

	logInfo(BCHAIN_LOG, "Received Block (%d), added to blockchain: (%s) createdBy (%s)", block.Number, block.Hash[:8], block.Author[:4])
}

func createBlock(blockCreation BlockCreation) Block {
	blockchainMutex.Lock()

	// Limit number of transactions per block
	if len(blockCreation.Transactions) > PROCESSED_TX_PER_BLOCK {
		blockCreation.Transactions = blockCreation.Transactions[:PROCESSED_TX_PER_BLOCK]
	}

	// Add block creation transaction to the block (outside mempool)
	blockCreation.Transactions = append(blockCreation.Transactions, createBlockRewardTx())

	var txHashes [][]byte
	for _, tx := range blockCreation.Transactions {
		txHash, err := hex.DecodeString(tx.Hash)
		if err != nil {
			logFatal(BCHAIN_LOG, "Error decoding transaction hash: %s", err)
		}
		txHashes = append(txHashes, txHash)
	}
	merkleRoot := computeMerkleRoot(txHashes)

	var previousBlockHash string
	var secondBlockHash string

	if blockCreation.ParentBlock != nil {
		previousBlockHash = blockCreation.ParentBlock.Hash
	} else {
		previousBlockHash = ""
	}

	if blockCreation.SecondParentBlock != nil {
		secondBlockHash = blockCreation.SecondParentBlock.Hash
	} else {
		secondBlockHash = ""
	}

	block := Block{
		Hash:                    "", // filled later from intial block fields
		Number:                  uint32(blockchainLength + 1),
		CreatedAt:               time.Now().Unix(),
		Transactions:            blockCreation.Transactions,
		MerkleRoot:              hex.EncodeToString(merkleRoot),
		Author:                  publicKey,
		PreviousBlock:           blockCreation.ParentBlock,
		PreviousBlockHash:       previousBlockHash,
		PreviousSecondBlock:     blockCreation.SecondParentBlock,
		PreviousSecondBlockHash: secondBlockHash,
		Depth:                   blockCreation.Depth,
		Row:                     blockCreation.Row,
		Col:                     blockCreation.Col,
		CommitmentHash:          blockCreation.CommitmentHash,
		CommitmentSecret:        blockCreation.CommitmentSecret,
		CommitmentSig:           blockCreation.CommitmentSecretSignature,
	}
	block.Hash = hex.EncodeToString(computeBlockHash(block, merkleRoot))

	// Append to blockchain
	blockchain = append(blockchain, block)
	blockchainLength++
	blockchainMutex.Unlock()

	// Remove transactions from the mempool
	mempoolMutex.Lock()
	for _, tx := range txHashes {
		delete(mempool, hex.EncodeToString(tx))
	}
	mempoolMutex.Unlock()

	logInfo(BCHAIN_LOG, "Created Block (%d) added to blockchain: (%s) createdBy (%s)", block.Number, block.Hash[:8], block.Author[:4])
	return block
}

func createBlockRewardTx() Transaction {
	tx := Transaction{
		Hash:      "",
		HashMask:  0,
		CreatedAt: time.Now().Unix(),
		Recipient: publicKey,
		Sender:    COINBASE_ADDRESS,
		Amount:    0,
	}

	tx.Hash = hex.EncodeToString(computeTransactionHash(tx))

	return tx
}

func computeTransactionHash(transaction Transaction) []byte {
	var data []byte
	data = append(data, []byte(transaction.Sender)...)
	data = append(data, []byte(transaction.Recipient)...)
	data = append(data, int64ToBytes(transaction.Amount)...)
	data = append(data, int64ToBytes(transaction.CreatedAt)...)

	return keccak256.New().Hash(data)
}

func computeBlockHash(block Block, merkleRoot []byte) []byte {
	var data []byte
	data = append(data, uInt32ToBytes(block.Number)...)
	data = append(data, merkleRoot...)
	data = append(data, []byte(block.Author)...)
	data = append(data, int64ToBytes(block.CreatedAt)...)

	return keccak256.New().Hash(data)
}

func computeMerkleRoot(txHashes [][]byte) []byte {
	tree, err := merkletree.NewTree(merkletree.WithData(txHashes))
	if err != nil {
		logError(BCHAIN_LOG, "Error creating Merkle tree: %s", err)
	}

	return tree.Root()
}

func validateBlockHash(block Block) bool {
	return hex.EncodeToString(computeBlockHash(block, []byte(block.MerkleRoot))) == block.Hash
}
