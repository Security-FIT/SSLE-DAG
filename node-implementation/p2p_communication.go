// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/wagslane/go-rabbitmq"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	"hash/fnv"
	"io"
	"maps"
	"math"
	"math/big"
	mathRand "math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NodeStreams struct {
	Streams map[string]StreamWrapper
}

type StreamWrapper struct {
	Buffer  *bufio.ReadWriter
	Stream  network.Stream
	Latency int64
}

var sendReadyMessageState bool
var sendReadyMessageMutex sync.Mutex

var messageHistory []string
var messagePeersHistory map[string][]string
var epochRoundsGenerated []uint32
var maxRandomMsgInteger = big.NewInt(32768)
var maxRandomCommitmentDelay = big.NewInt(350)
var messagePeersHistoryMutex sync.Mutex

var randomJitterGenerator *mathRand.Rand

var mutex sync.Mutex
var histMutex sync.Mutex
var epochMutex sync.Mutex
var commitmentBucketMutex sync.Mutex
var submittedCommitmentsMutex sync.Mutex

var commitmentGenEpochRound uint32
var commitmentRoundMutex sync.Mutex

// Epoch-dev variables
var submittedCommitments []CommitmentInfo
var commitmentsBucket []string
var commitmentsBucketMap map[string]CommitmentSharedInfo
var bucketHash string
var bucketPositions []int

// Epoch-ready variables
var rdySubmittedCommitments []CommitmentInfo
var rdyCommitmentsBucket []string
var rdyCommitmentsBucketMap map[string]CommitmentSharedInfo
var rdyBucketHash string
var rdyBucketPositions []int
var rdyEpochRound uint32

// Epoch-running variables
var runSubmittedCommitments []CommitmentInfo
var runCommitmentsBucket []string
var runCommitmentsBucketMap map[string]CommitmentSharedInfo
var runBucketHash string
var runBucketPositions []int
var runEpochRound uint32

var bucketInfoStarted bool
var bucketInfoMutex sync.Mutex
var bucketHashesCheck []string
var bucketHashesCheckMutex sync.Mutex

var timeSynced bool
var syncedTime int64
var timeSyncNodes []int64
var timeSyncNodesMutex sync.Mutex

var exchangedPubKey []string
var exchangedPubKeyMutex sync.Mutex
var exchangedPubKeySentInitialMessage bool

var pubKeysMerkleLeaves [][]byte
var leafHashes [][]byte
var levels [][][]byte
var pubMerkleTreeRoot []byte
var pathNodes [PUB_KEY_MERKLE_TREE_DEPTH][]byte
var pathIndices [PUB_KEY_MERKLE_TREE_DEPTH]uint8
var pubKeyLeafIndex int

func handleStream(stream network.Stream) {
	// Get peer id from the protocol name as this function cannot take any extra arguments and global variable can be problematic due to concurrency
	peerId := strings.Split(string(stream.Protocol()), "_")[1]

	logInfo(P2P_LOG, "Got a new stream from: %s", peerId)

	// Create a buffer stream for non-blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	// Set stream and buffer for the peer
	streamWrapper := nodeStreams.Streams[peerId]
	streamWrapper.Buffer = rw
	streamWrapper.Stream = stream
	nodeStreams.Streams[peerId] = streamWrapper

	go readData(rw)

	if len(nodeStreams.Streams) == expectedNumberOfPeers {
		sendRabbitMqReadyMessage()
	}
}

/**
 * Read data from one opened stream and pass a message in structured format to the processMessage function
 */
func readData(rw *bufio.ReadWriter) {
	for {
		// Every message is delimitered by a newline
		str, err := rw.ReadString('\n')

		if err != nil {
			if err != io.EOF {
				logError(P2P_LOG, "Read error: %s", err)
			}
			return
		}

		if str == "" {
			logError(P2P_LOG, "Empty message")
			return
		}

		// Remove newline character in the end
		data := str[:len(str)-1]

		logDebug(P2P_LOG, "Received message: %s", data)

		// Parse the incoming message
		networkMessage := new(NetworkMessage)
		err = json.Unmarshal([]byte(data), networkMessage)
		if err != nil {
			logError(P2P_LOG, "Unmarshal consensus message error: %s", err)
			continue
		}

		receiverId := getIdForBufioStream(rw)

		histMutex.Lock()

		// Store peer from whom received the message
		messagePeersHistoryMutex.Lock()
		messagePeersHistory[receiverId] = append(messagePeersHistory[receiverId], networkMessage.Id)
		messagePeersHistoryMutex.Unlock()

		// Check if the message is intended to be gossiped (is not unicast or gossiped before)
		if isMessageFirstOccurrence(networkMessage.Id) {
			// Store message in history of sent/gossiped messages
			messageHistory = append(messageHistory, networkMessage.Id)

			// Remove the oldest half of the messages if the history cross certain threshold
			//if len(messageHistory) > MAX_MESSAGE_HISTORY_SIZE {
			//	messageHistory = messageHistory[MAX_MESSAGE_HISTORY_SIZE/2:]
			//}
			histMutex.Unlock()

			for peerId, streamWrapper := range nodeStreams.Streams {
				go gossipMessage(streamWrapper, peerId, data, networkMessage.Id)
			}

			processNetworkMessage(*networkMessage)
		} else {
			histMutex.Unlock()
		}
	}
}

/**
 * Send message to all nodes
 */
func sendMessageBroadcast(msgType Message, block BlockTransfer, commitment string, epochRound uint32, extraInt int64) NetworkMessage {
	randInt, err := rand.Int(rand.Reader, maxRandomMsgInteger)
	if err != nil {
		logFatal(P2P_LOG, "Failed to generate random number for message ID: %s", err)
	}
	randId := randInt.String()
	timeString := strconv.FormatInt(time.Now().UnixMicro(), 10)

	var builder strings.Builder
	builder.WriteString(randId)
	builder.WriteString("_")
	builder.WriteString(timeString)
	message := NetworkMessage{
		Id:                   builder.String(),
		Type:                 msgType,
		BlockTransfer:        block,
		Commitment:           commitment,
		CommitmentEpochRound: epochRound,
		ExtraInt:             extraInt,
	}
	messageJson, err := json.Marshal(message)
	if err != nil {
		logFatal(P2P_LOG, "Failed to marshal message during broadcast: %s", err)
	}

	// Store message to not receive it back. Use mutex as append is not thread-safe
	histMutex.Lock()
	messageHistory = append(messageHistory, message.Id)
	histMutex.Unlock()

	for peerId, stream := range nodeStreams.Streams {
		go gossipMessage(stream, peerId, string(messageJson), message.Id)
	}

	return message
}

/**
 * Send message to only one specific node
 */
func sendMessageUnicast(msgType Message, peerId string, block BlockTransfer, commitment string, epochRound uint32, extraInt int64) {
	mutex.Lock()

	message := NetworkMessage{
		Id:                   strconv.FormatInt(time.Now().UnixMicro(), 10),
		Type:                 msgType,
		BlockTransfer:        block,
		Commitment:           commitment,
		CommitmentEpochRound: epochRound,
		ExtraInt:             extraInt,
	}

	_, err := nodeStreams.Streams[peerId].Buffer.WriteString(fmt.Sprintf("%s\n", message))
	if err != nil {
		logFatal(P2P_LOG, "Failed to write message to the buffer: %s", err)
	}

	mutex.Unlock()
}

/**
 * Process received message
 */
func processNetworkMessage(message NetworkMessage) {
	switch message.Type {
	case BCHAIN_INIT_COMMITMENT, BCHAIN_COMMITMENT:
		if message.Type != BCHAIN_INIT_COMMITMENT {
			// Validate commitment
			validCommitment, publicWitness := isCommitmentValid(message.Commitment, commitmentGenEpochRound)

			if !validCommitment {
				logError(P2P_LOG, "Invalid commitment")
				return
			} else {
				commitmentBucketMutex.Lock()
				commitmentHash := hex.EncodeToString(keccak256.New().Hash([]byte(message.Commitment)))
				commitmentSharedInfo := CommitmentSharedInfo{
					Commitment: message.Commitment,
					Hash:       commitmentHash,
					Public:     publicWitness,
				}

				// Check if already received the commitmentHash
				for i := len(commitmentsBucket) - 1; i >= 0; i-- {
					if commitmentsBucket[i] == commitmentHash {
						commitmentBucketMutex.Unlock()
						return
					}
				}

				commitmentsBucketMap[commitmentHash] = commitmentSharedInfo

				commitmentsBucket = append(commitmentsBucket, commitmentHash)

				if len(commitmentsBucket) == MAX_COMMITMENT_VAR*numOfNodes {
					sort.Strings(commitmentsBucket)
				}
				commitmentBucketMutex.Unlock()

				commitmentRoundMutex.Lock()
				if message.CommitmentEpochRound != commitmentGenEpochRound {
					logError(P2P_LOG, "Received commitment for the wrong epoch round (%d), expected (%d)", message.CommitmentEpochRound, commitmentGenEpochRound)
					return
				}
				commitmentRoundMutex.Unlock()

				// Send info to the RabbitMQ
				initCommitmentMessage := RMQMessageCommitment{
					Type:          BCHAIN_COMMITMENT,
					NodeId:        "-1", // Intentionally unspecified or unknown node
					Commitment:    message.Commitment,
					PublicWitness: publicWitness,
					EpochRound:    message.CommitmentEpochRound,
				}
				sendRabbitMQMessage(initCommitmentMessage)
			}
		}

		// At this point, commitmentGenEpochRound == message.CommitmentEpochRound

		// Skip generation if commitment for this epoch round is already generated
		epochMutex.Lock()
		for i := len(epochRoundsGenerated) - 1; i >= 0; i-- {
			if epochRoundsGenerated[i] == message.CommitmentEpochRound {
				epochMutex.Unlock()
				return
			}
		}
		epochMutex.Unlock()

		// Mark commitment for this epoch round as generated
		epochMutex.Lock()
		epochRoundsGenerated = append(epochRoundsGenerated, message.CommitmentEpochRound)

		// Remove old rounds to reduce memory usage
		//if len(epochRoundsGenerated) > MAX_EPOCH_ROUNDS_SIZE {
		//	epochRoundsGenerated = epochRoundsGenerated[MAX_EPOCH_ROUNDS_SIZE/2:]
		//}
		epochMutex.Unlock()

		go func() {
			// Commitment was not generated for this epoch round, generate it
			for i := MIN_COMMITMENT_VAR; i <= MAX_COMMITMENT_VAR; i++ {
				// Intentionally create delay for multiple commitment generation to avoid identity information leakage
				// Delay can be modified by individual nodes to add more randomness
				randInt, err := rand.Int(rand.Reader, maxRandomCommitmentDelay)
				if err != nil {
					logError(P2P_LOG, "Failed to generate random number for commitment delay: %s", err)
					return
				}
				time.Sleep(time.Millisecond * time.Duration(randInt.Int64())) // Milliseconds * max(maxRandomCommitmentDelay) in nanoseconds
				publicWitness, proof, secret, secretSig := generateCommitment(message.CommitmentEpochRound, uint64(i))

				// Encode commitment into two base64 strings
				commitment := commitmentBase64Encode(publicWitness, proof)
				commitmentHash := hex.EncodeToString(keccak256.New().Hash([]byte(commitment)))

				// Store created commitment locally
				submittedCommitmentsMutex.Lock()
				commitmentInfo := CommitmentInfo{
					Commitment: commitment,
					Hash:       commitmentHash,
					Secret:     secret,
					SecretSig:  secretSig,
				}
				submittedCommitments = append(submittedCommitments, commitmentInfo)
				submittedCommitmentsMutex.Unlock()

				// Store commitment in the bucket
				commitmentBucketMutex.Lock()
				commitmentSharedInfo := CommitmentSharedInfo{
					Commitment: commitment,
					Hash:       commitmentHash,
					Public:     PublicWitness{}, // Can be nil because there is no need to validate block with commitment that I have submitted
				}
				commitmentsBucketMap[commitmentHash] = commitmentSharedInfo
				logInfo(P2P_LOG, "Commitment (%d/%d) for epoch (%d) generated: (%s)", i, MAX_COMMITMENT_VAR, message.CommitmentEpochRound, commitmentHash[:8])

				commitmentsBucket = append(commitmentsBucket, commitmentHash)

				if len(commitmentsBucket) == MAX_COMMITMENT_VAR*numOfNodes {
					randomBucketShuffle(false)
					bucketHash, bucketPositions = identifyCommitmentPositions()
				}
				commitmentBucketMutex.Unlock()

				sendMessageBroadcast(BCHAIN_COMMITMENT, BlockTransfer{}, commitment, message.CommitmentEpochRound, 0)

				// Send info to the RabbitMQ
				initCommitmentMessage := RMQMessageCommitment{
					Type:          BCHAIN_COMMITMENT,
					NodeId:        selfIdStr,
					Commitment:    commitment,
					EpochRound:    message.CommitmentEpochRound,
					PublicWitness: PublicWitness{},
				}

				sendRabbitMQMessage(initCommitmentMessage)
			}

			// Prepare for initial bucket sorting
			if len(blockchain) == 1 {
				time.Sleep(time.Millisecond * 1000 * 4)

				bucketInfoMessage := NetworkMessage{
					Id:                   "000000", // ID of the message will be skipped,
					Type:                 BCHAIN_INIT_BUCKET_INFO,
					BlockTransfer:        BlockTransfer{},
					Commitment:           selfIdStr,
					CommitmentEpochRound: commitmentGenEpochRound,
					ExtraInt:             int64(len(commitmentsBucket)),
				}
				processNetworkMessage(bucketInfoMessage)
				// sendMessageBroadcast(BCHAIN_INIT_BUCKET_INFO, BlockTransfer{}, selfIdStr, commitmentGenEpochRound, int64(len(commitmentsBucket)))
			}
		}()

		break
	case BCHAIN_INIT_BUCKET_INFO:
		commitmentBucketMutex.Lock()
		initializeBucketSort()
		commitmentBucketMutex.Unlock()

		if commitmentGenEpochRound < 2 {
			commitmentBucketMutex.Lock()
			randomBucketShuffle(true)
			bucketHash, bucketPositions = identifyCommitmentPositions()
			commitmentBucketMutex.Unlock()

			bucketInfoMessage := RMQMessageBucketInfo{
				Type:              BCHAIN_INIT_BUCKET_INFO,
				NodeId:            selfIdStr,
				BucketLength:      len(commitmentsBucket),
				BucketHash:        bucketHash,
				SelectedPositions: bucketPositions,
			}
			sendRabbitMQMessage(bucketInfoMessage)

			bucketInfoMutex.Lock()
			if !bucketInfoStarted {
				bucketInfoStarted = true
				bucketInfoMutex.Unlock()

				// Store data to broadcast to avoid concurrency issues
				broadcastBucketHash := bucketHash
				broadcastCommitmentGenEpochRound := commitmentGenEpochRound
				broadcastBucketLength := int64(len(commitmentsBucket))

				time.Sleep(time.Millisecond * 4000)

				sendMessageBroadcast(BCHAIN_BUCKET_CHECK, BlockTransfer{Author: selfIdStr}, broadcastBucketHash, broadcastCommitmentGenEpochRound, broadcastBucketLength)
				logInfo(P2P_LOG, "Broadcasting BUCKET_CHECK message from (%s) with bucket length (%d)", selfIdStr, broadcastBucketLength)
				bucketInfoMutex.Lock()
				bucketInfoStarted = false
				bucketInfoMutex.Unlock()
			} else {
				bucketInfoMutex.Unlock()
			}
		}
		break
	case BCHAIN_BUCKET_CHECK:
		receivedBucketHash := message.Commitment
		receivedBucketLength := message.ExtraInt

		invalidHash := false
		// Check if the received bucket hash is equal to the local bucket hash

		if commitmentGenEpochRound == message.CommitmentEpochRound && receivedBucketHash != bucketHash {
			logError(P2P_LOG, "Received bucket hash (%s) is not equal to the local bucket hash (%s), local length (%d)", receivedBucketHash, bucketHash, len(commitmentsBucket))
			logError(P2P_LOG, "Local commitmentEpochRound (%d), received commitmentEpochRound (%d)", commitmentGenEpochRound, message.CommitmentEpochRound)

			// second try
			logInfoBold(P2P_LOG, "Trying to fix bucket hash...")

			bucketHashesCheckMutex.Lock()
			commitmentBucketMutex.Lock()
			randomBucketShuffle(false)
			bucketHash, bucketPositions = identifyCommitmentPositions()
			commitmentBucketMutex.Unlock()
			bucketHashesCheckMutex.Unlock()

			sendMessageBroadcast(BCHAIN_BUCKET_CHECK, BlockTransfer{}, bucketHash, commitmentGenEpochRound, int64(len(commitmentsBucket)))
			return
		}

		if receivedBucketLength != int64(len(commitmentsBucket)) {
			logError(P2P_LOG, "Received bucket length (%d) is not equal to the local bucket length (%d)", receivedBucketLength, len(commitmentsBucket))
			invalidHash = true
		}

		if invalidHash {
			return
		}

		bucketHashesCheckMutex.Lock()
		bucketHashesCheck = append(bucketHashesCheck, receivedBucketHash)

		if len(bucketHashesCheck) == numOfOtherNodes {
			logInfo(P2P_LOG, "Has all (%d) bucket hashes", len(bucketHashesCheck))
			matchedHashes := 0
			// Check what is the percentage match of received hashes
			for _, foundHash := range bucketHashesCheck {
				if foundHash == bucketHash {
					matchedHashes++
				}
			}

			fracMatched := float64(matchedHashes) / float64(numOfOtherNodes)

			logInfo(P2P_LOG, "Bucket hash check for epoch (%d) has (%.1f%%) match", commitmentGenEpochRound, fracMatched*100)

			// Send info to RabbitMQ
			rmqBucketCheck := RMQMessageBucketCheck{
				Type:   BCHAIN_BUCKET_CHECK,
				NodeId: selfIdStr,
				Match:  fracMatched,
			}
			sendRabbitMQMessage(rmqBucketCheck)

			if fracMatched < 0.5 {
				logError(P2P_LOG, "Consensus match is lower than 50%%, bucket hash check for epoch (%d) failed", commitmentGenEpochRound)
				bucketHashesCheckMutex.Unlock()
				//return
			}
			bucketHashesCheckMutex.Unlock()

			if commitmentGenEpochRound == 0 {
				// Pass commitment vars to epoch-ready vars
				passToEpochReady()

				time.Sleep(time.Millisecond * 4000)

				// Start directly processing the next commitment round
				nextCommitmentRoundMessage := NetworkMessage{
					Id:                   "000000", // ID of the message will be skipped,
					Type:                 BCHAIN_INIT_COMMITMENT,
					BlockTransfer:        BlockTransfer{},
					Commitment:           "",
					CommitmentEpochRound: commitmentGenEpochRound,
					ExtraInt:             0,
				}
				processNetworkMessage(nextCommitmentRoundMessage)
			} else if commitmentGenEpochRound == 1 {
				// Pass commitment vars to epoch-running vars
				passToEpochRunning()

				time.Sleep(time.Millisecond * 4000)

				// Pass commitment vars to epoch-ready vars
				passToEpochReady()

				/*
					Sync time with other nodes for block generation on the closest [x+1:00], where:
					- x+1 stands for current time minutes + 1
					- 00 stands for fixed 0 seconds

					If current time seconds are more than 45, then either:
					- wait 5 seconds to adapt to the closest x+1:00 Second time
					- receive time sync from other nodes to x+1:00 and adapt

					This time indicates the time of publishing of the 1st block in the epoch
				*/
				syncTime := time.Now()
				if syncTime.Second() < 45 {
					syncTime = syncTime.Add(time.Minute).Truncate(time.Minute)
				} else {
					// To avoid possible collision of multiple times to sync right before and after 45th Second
					// Wait 5 seconds to adapt to the closest x+1:00 Second time, if some node announces it
					logDebug(P2P_LOG, "Waiting for time sync to x+1:00 Second...")
					time.Sleep(time.Second * 5)

					timeSyncNodesMutex.Lock()
					if timeSynced {
						logDebug(P2P_LOG, "Time is already synced, skipping new time sync")
						timeSyncNodesMutex.Unlock()
						return
					} else {
						logDebug(P2P_LOG, "Time is not yet synced, syncing now...")
						timeSyncNodesMutex.Unlock()
						syncTime = time.Now()
						syncTime = syncTime.Add(time.Minute).Truncate(time.Minute)
					}
				}

				syncedTime = syncTime.Unix()
				sendMessageBroadcast(BCHAIN_SYNC_TIME, BlockTransfer{}, "", 0, syncTime.Unix())
			} else {
				// Pass commitment vars to epoch-running vars
				passToEpochRunning()

				// Pass commitment vars to epoch-ready vars
				passToEpochReady()
			}
		} else {
			logInfo(P2P_LOG, "Has (%d) bucket hashes, waiting to receive (%d)", len(bucketHashesCheck), numOfOtherNodes)
			bucketHashesCheckMutex.Unlock()
		}

		bucketHashesCheckMutex.Lock()
		if len(bucketHashesCheck) > numOfOtherNodes {
			logError(P2P_LOG, "Received bucket hash check from more nodes (%d) than expected (%d)", len(bucketHashesCheck), numOfOtherNodes)
			bucketHashesCheckMutex.Unlock()
			return
		}
		bucketHashesCheckMutex.Unlock()

		break
	case BCHAIN_SYNC_TIME:
		timeSyncNodesMutex.Lock()
		if timeSynced {
			logError(P2P_LOG, "Time is already synced, skipping new time sync")
			timeSyncNodesMutex.Unlock()
			return
		}

		timeSyncNodes = append(timeSyncNodes, message.ExtraInt)

		if len(timeSyncNodes) == numOfOtherNodes {
			matchedTimes := 0
			// Check what is the percentage match of received times
			for _, timeToSync := range timeSyncNodes {
				if timeToSync == syncedTime {
					matchedTimes++
				}
			}

			fracMatched := float64(matchedTimes) / float64(numOfOtherNodes)

			unixTime := time.Unix(syncedTime, 0)
			logInfo(P2P_LOG, "Time to publish 1st block is synced to (%s) with (%.1f%%) match", unixTime.Format(time.DateTime), fracMatched*100)

			if fracMatched < 0.5 {
				logError(P2P_LOG, "Consensus match is lower than 50%%, time sync failed")
				logError(P2P_LOG, "Adapting to the rest of the network...")

				majorElement, success := findMajorElementInt64(timeSyncNodes)

				if !success {
					logFatal(P2P_LOG, "Failed to find major element in the time sync nodes in: %v", timeSyncNodes)
				} else {
					syncedTime = majorElement
					logInfo(P2P_LOG, "Time to publish 1st block is synced to (%s)", unixTime.Format(time.DateTime))
				}
			}

			timeSynced = true

			go blockGenerationSchedule()

			// Send info to RabbitMQ
			rmqBucketCheck := RMQMessageTimeSync{
				Type:     BCHAIN_BUCKET_CHECK,
				NodeId:   selfIdStr,
				SyncTime: syncedTime,
			}
			sendRabbitMQMessage(rmqBucketCheck)
		} else {
			logDebug(P2P_LOG, "Has (%d) time syncs, waiting to receive (%d)", len(timeSyncNodes), numOfOtherNodes)
		}

		if len(timeSyncNodes) > numOfOtherNodes {
			logError(P2P_LOG, "Received time syncs from more nodes (%d) than expected (%d)", len(timeSyncNodes), numOfOtherNodes)
			timeSyncNodesMutex.Unlock()
			return
		}
		timeSyncNodesMutex.Unlock()

		break
	case BCHAIN_BLOCK:
		logInfo(P2P_LOG, "Received block: (%s...)", message.BlockTransfer.Hash[:8])

		// Find commitment associated with the block
		sharedCommitmentInfo, commitmentFound := runCommitmentsBucketMap[message.BlockTransfer.CommitmentHash]

		if !commitmentFound {
			logError(P2P_LOG, "Commitment for the block (%s...) not found", message.BlockTransfer.Hash[:8])
			return
		}

		// On success, process the block
		processBlock(message.BlockTransfer, sharedCommitmentInfo)

		if (blockchainLength-1)%EPOCH_SIZE < EPOCH_SIZE/2 {
			//commitmentBucketMutex.Lock()
			//randomBucketShuffle(false)
			//bucketHash, bucketPositions = identifyCommitmentPositions()
			//commitmentBucketMutex.Unlock()
		}

		// Skip sending broadcasted  block to RabbitMQ to have less overhead
		//blockMessage := RMQMessageBlockBuild{
		//	Type:   BCHAIN_BLOCK,
		//	NodeId: "-1",
		//	Block:  message.BlockTransfer,
		//}
		//sendRabbitMQMessage(blockMessage)
		break
	case PING:
		pingMessage := RMQMessagePing{
			Id:       message.Id,
			Type:     message.Type,
			NodeId:   selfIdStr,
			NodeTime: time.Now().UnixMilli(),
		}
		sendRabbitMQMessage(pingMessage)
		break
	case GENESIS_BLOCK_BUILD:
		genesisMessageData := RMQMessageGenesisBlockBuild{
			Type:      message.Type,
			NodeId:    selfIdStr,
			PublicKey: publicKey,
		}
		sendRabbitMQMessage(genesisMessageData)
	case INFO:
		break
	case TRANSACTION_PACK:
		break
	case PUB_KEY_EXCHANGE:
		receivedPublicKey := message.Commitment

		sendOwnPubKey := false

		exchangedPubKeyMutex.Lock()
		if len(exchangedPubKey) == 1 {
			sendOwnPubKey = true
		}

		pubKeyFound := false
		for _, key := range exchangedPubKey {
			if key == receivedPublicKey {
				pubKeyFound = true
				break
			}
		}

		if !pubKeyFound {
			exchangedPubKey = append(exchangedPubKey, receivedPublicKey)
		}

		exchangedPubKeyMutex.Unlock()
		if sendOwnPubKey && !exchangedPubKeySentInitialMessage {
			sendMessageBroadcast(PUB_KEY_EXCHANGE, BlockTransfer{}, getPublicKey(), 0, 0)
		}

		if len(exchangedPubKey) == numOfNodes {
			logInfo(P2P_LOG, "Received all public keys for merkle tree")

			exchangedPubKeyMutex.Lock()
			sort.Strings(exchangedPubKey)
			exchangedPubKeyMutex.Unlock()

			for i := 0; i < PUB_KEY_MERKLE_TREE_SIZE; i++ {
				if i < len(exchangedPubKey) {
					pubKeysMerkleLeaves = append(pubKeysMerkleLeaves, []byte(exchangedPubKey[i]))
				} else {
					pubKeysMerkleLeaves = append(pubKeysMerkleLeaves, []byte("0"))
				}
			}

			hasher := hash.MIMC_BLS12_381.New()

			// Compute leaf hashes
			for i, v := range pubKeysMerkleLeaves {
				hasher.Reset()
				_, e := hasher.Write(v)
				if e != nil {
					logError(P2P_LOG, "Failed to compute leaf hash: %s", e)
				}

				leafHashes[i] = hasher.Sum(nil)
			}

			// Build Merkle tree levels
			levels = [][][]byte{leafHashes}
			for i, v := range leafHashes {
				levels[0][i] = v
			}
			for level := 1; level <= PUB_KEY_MERKLE_TREE_DEPTH; level++ {
				prev := levels[level-1]
				curr := make([][]byte, len(prev)/2)
				for i := 0; i < len(prev); i += 2 {
					hasher.Reset()
					_, e := hasher.Write(prev[i])
					if e != nil {
						logError(P2P_LOG, "Failed to compute hash for level %d: %s", level, e)
					}

					_, e = hasher.Write(prev[i+1])
					if e != nil {
						logError(P2P_LOG, "Failed to compute hash for level %d: %s", level, e)
					}

					curr[i/2] = hasher.Sum(nil)
				}
				levels = append(levels, curr)
			}
			pubMerkleTreeRoot = levels[PUB_KEY_MERKLE_TREE_DEPTH][0]

			// Find index of mine public key
			for i, key := range exchangedPubKey {
				if key == getPublicKey() {
					pubKeyLeafIndex = i
					break
				}
			}

			// Collect path nodes and index bits
			idx := pubKeyLeafIndex
			for i := 0; i < PUB_KEY_MERKLE_TREE_DEPTH; i++ {
				siblingIndex := idx ^ 1 // flip last bit
				pathNodes[i] = levels[i][siblingIndex]
				pathIndices[i] = uint8(idx & 1)
				idx >>= 1
			}

			logInfo(P2P_LOG, "Computed merkle tree proof path from public key merkle tree")

			pubKeysSynced := RMQMessagePublicKeysSynced{
				Type:   message.Type,
				NodeId: selfIdStr,
			}
			sendRabbitMQMessage(pubKeysSynced)
		}
		break
	default:
		break
	}
}

func blockGenerationSchedule() {
	logInfo(P2P_LOG, "Starting block generation schedule...")

	go func() {
		for {
			// Wait for the next block slot
			waitMilliseconds := 1000*(syncedTime+(absRound*int64(BLOCK_TIME_SECONDS))) - time.Now().UnixMilli()
			logInfo(P2P_LOG, "Loop wait time (%d) seconds", waitMilliseconds/1000)
			time.Sleep(time.Millisecond * time.Duration(waitMilliseconds))

			var epochIndexes []int
			var epochActions []DagActionType
			var filteredEpochIndexes []int
			var filteredEpochActions []DagActionType
			var filteredOriginalIndexMapping []int
			for i := 0; i < DAG_PARALLEL_LEN; i++ {
				// Store and check all parallel options in case a merge is found and
				// another leader does not create a block for closing branch
				epochIndexes = append(epochIndexes, int(slot)+i)
			}

			// Decide action for the index
			for _, idx := range epochIndexes {
				action := decideAction(idx%DAG_PARALLEL_LEN, int(absRound)+1)
				epochActions = append(epochActions, action)
			}

			for idx, action := range epochActions {
				if action.Action == DAG_ACTION_MERGE {
					// Skip blocks for the finished branch
					epochActions[action.Second.Row].Action = DAG_ACTION_SKIP
				} else if action.Action == DAG_ACTION_SPLIT {
					// Add special mark to second block of split
					epochActions[getSplitSecondIndex(idx, action.Parent.Depth+1)].Action = DAG_ACTION_SPLIT_SND
					epochActions[getSplitSecondIndex(idx, action.Parent.Depth+1)].Parent = action.Parent
				}
			}

			var expectedNumOfNewBlocks = 0
			// Filter currently assigned slots to the node that also don't have SKIP action
			for i, action := range epochActions {
				if action.Action != DAG_ACTION_SKIP {
					expectedNumOfNewBlocks++
					if sliceContainsInt(runBucketPositions, epochIndexes[i]) {
						filteredEpochIndexes = append(filteredEpochIndexes, epochIndexes[i]%DAG_PARALLEL_LEN)
						filteredEpochActions = append(filteredEpochActions, action)
						filteredOriginalIndexMapping = append(filteredOriginalIndexMapping, epochIndexes[i])
					}
				}
			}

			logInfoBold(P2P_LOG, "Expected number of new blocks: %d at epoch %d, slots (%d-%d)", expectedNumOfNewBlocks, commitmentGenEpochRound, slot, slot+int64(DAG_PARALLEL_LEN)-1)
			logInfoBold(P2P_LOG, "Generating %d (blocks) at absolute round (%d)", len(filteredEpochIndexes), absRound)

			// ================================================================================
			logInfoBold(P2P_LOG, "Actions for me:")
			for _, action := range filteredEpochActions {
				switch action.Action {
				case DAG_ACTION_SPLIT:
					logInfoBold(P2P_LOG, "Action DAG_ACTION_SPLIT")
					break
				case DAG_ACTION_SPLIT_SND:
					logInfoBold(P2P_LOG, "Action DAG_ACTION_SPLIT_SND")
					break
				case DAG_ACTION_MERGE:
					logInfoBold(P2P_LOG, "Action DAG_ACTION_MERGE")
					break
				case DAG_ACTION_CONTINUE:
					logInfoBold(P2P_LOG, "Action DAG_ACTION_CONTINUE")
					break
				case DAG_ACTION_SKIP:
					logInfoBold(P2P_LOG, "Action DAG_ACTION_SKIP")
					break
				}
			}
			// ================================================================================

			for i, indexInEpoch := range filteredEpochIndexes {
				var includedTransactions []Transaction
				var depth int

				switch filteredEpochActions[i].Action {
				case DAG_ACTION_CONTINUE:
					depth = filteredEpochActions[i].Parent.Depth
					break
				case DAG_ACTION_SPLIT, DAG_ACTION_SPLIT_SND:
					depth = filteredEpochActions[i].Parent.Depth + 1
					break
				case DAG_ACTION_MERGE:
					depth = filteredEpochActions[i].Parent.Depth - 1
					break
				default:
					logError(P2P_LOG, "Unknown action for block creation", filteredEpochActions[i].Action)
					break
				}

				mempoolMutex.Lock()
				for _, tx := range mempool {
					if doesTxBelongToIndex(*tx, depth, indexInEpoch) {
						includedTransactions = append(includedTransactions, *tx)
					}
				}
				mempoolMutex.Unlock()

				commitmentInfo := getCommitmentInfoByHash(runCommitmentsBucket[filteredOriginalIndexMapping[i]])

				blockCreation := BlockCreation{
					CommitmentHash:            commitmentInfo.Hash,
					CommitmentSecret:          commitmentInfo.Secret,
					CommitmentSecretSignature: commitmentInfo.SecretSig,
					Transactions:              includedTransactions,
					Depth:                     depth,
					ParentBlock:               filteredEpochActions[i].Parent,
					SecondParentBlock:         filteredEpochActions[i].Second,
					Row:                       indexInEpoch,
					Col:                       int(absRound) + 1,
				}
				newBlock := createBlock(blockCreation)

				newBlockTransfer := BlockTransfer{
					Hash:                    newBlock.Hash,
					Number:                  newBlock.Number,
					CreatedAt:               newBlock.CreatedAt,
					Transactions:            newBlock.Transactions,
					MerkleRoot:              newBlock.MerkleRoot,
					Author:                  newBlock.Author,
					PreviousBlockHash:       newBlock.PreviousBlockHash,
					PreviousSecondBlockHash: newBlock.PreviousSecondBlockHash,
					Depth:                   newBlock.Depth,
					Row:                     newBlock.Row,
					Col:                     newBlock.Col,
					CommitmentHash:          newBlock.CommitmentHash,
					CommitmentSecret:        newBlock.CommitmentSecret,
					CommitmentSig:           commitmentInfo.SecretSig,
				}

				sendMessageBroadcast(BCHAIN_BLOCK, newBlockTransfer, "", 0, 0)

				//if (blockchainLength-1)%EPOCH_SIZE < EPOCH_SIZE/2 {
				//	randomBucketShuffle(false)
				//}

				blockMessage := RMQMessageBlockBuild{
					Type:   BCHAIN_BLOCK,
					NodeId: selfIdStr,
					Block:  newBlockTransfer,
				}
				sendRabbitMQMessage(blockMessage)
			}

			slot = slot + int64(DAG_PARALLEL_LEN)
			absRound++

			if slot%int64(EPOCH_SIZE) == int64(DAG_PARALLEL_LEN)*2 {
				// Prepare next commitments in the middle of the epoch

				// Random sorting is skipped in the slot of commitment generation
				nextCommitmentRoundMessage := NetworkMessage{
					Id:                   "000000", // ID of the message will be skipped,
					Type:                 BCHAIN_INIT_COMMITMENT,
					BlockTransfer:        BlockTransfer{},
					Commitment:           "",
					CommitmentEpochRound: commitmentGenEpochRound,
					ExtraInt:             0,
				}
				processNetworkMessage(nextCommitmentRoundMessage)
			}

			if (slot%int64(EPOCH_SIZE)) == 0 && slot > 0 {
				// Prepare next epoch in last slot

				// Sleep to avoid possible collision with extra blocks
				logInfoBold(P2P_LOG, "Size of generate commitments at round (%d) is %d", absRound, len(submittedCommitments))
				time.Sleep(time.Millisecond * 1000)

				slot = 0

				bucketHashesCheckMutex.Lock()
				commitmentBucketMutex.Lock()
				randomBucketShuffle(false)
				bucketHash, bucketPositions = identifyCommitmentPositions()
				bucketHashesCheckMutex.Unlock()
				commitmentBucketMutex.Unlock()

				bucketInfoMutex.Lock()
				if !bucketInfoStarted {
					bucketInfoStarted = true
					bucketInfoMutex.Unlock()

					sendMessageBroadcast(BCHAIN_BUCKET_CHECK, BlockTransfer{}, bucketHash, commitmentGenEpochRound, int64(len(commitmentsBucket)))
					bucketInfoMutex.Lock()
					bucketInfoStarted = false
					bucketInfoMutex.Unlock()
				} else {
					bucketInfoMutex.Unlock()
				}
			}
		}
	}()
}

func sendRabbitMQMessage(message any) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		logError(P2P_LOG, "Cannot create RabbitMQ message: %s", err)
	}

	err = rmqBlockchainPublisher.Publish(jsonMsg,
		[]string{"blockchain_data"},
		rabbitmq.WithPublishOptionsContentType("application/json"))
	if err != nil {
		logError(P2P_LOG, "Cannot publish RabbitMQ message: %s", err)
	}
}

func gossipMessage(streamWrapper StreamWrapper, peerId string, messageSerialized string, messageId string) {
	// Simulate the connection latency for a specific node
	logDebug(P2P_LOG, "Simulate latency: %d", streamWrapper.Latency/1_000_000)
	time.Sleep((time.Duration(streamWrapper.Latency) + time.Duration(generateJitter())) * time.Nanosecond) // Latency is stored in nanoseconds to avoid float precision loss
	logDebug(P2P_LOG, "End of latency")

	// Lock the mutex to prevent concurrent writes to the stream
	mutex.Lock()

	// Optimization - if peer already received this message do not send it to him
	if messageAlreadyReceived(peerId, messageId) {
		mutex.Unlock()
		return
	}

	logDebug(P2P_LOG, "Gossiping message: %s", messageSerialized)

	// Node needs to store from whom received the message so it doesn't send it back to the sender (doesn't make sense), use the same structure as for rabbitmq optimization

	_, err = streamWrapper.Buffer.WriteString(fmt.Sprintf("%s\n", messageSerialized))
	if err != nil {
		logFatal(P2P_LOG, "Failed to write message to the buffer: %s", err)
	}

	err = streamWrapper.Buffer.Flush()
	if err != nil {
		logFatal(P2P_LOG, "Failed to flush the buffer: %s", err)
	}

	// Unlock the mutex
	mutex.Unlock()
}

func isMessageFirstOccurrence(messageId string) bool {
	// Iterate from the end because the most recent messages are at the end
	for i := len(messageHistory) - 1; i >= 0; i-- {
		// If the message is already in the history, do not gossip it
		if messageHistory[i] == messageId {
			return false
		}
	}
	return true
}

func messageAlreadyReceived(peerId string, messageId string) bool {
	messagePeersHistoryMutex.Lock()
	for i := len(messagePeersHistory[peerId]) - 1; i >= 0; i-- {
		if messagePeersHistory[peerId][i] == messageId {
			messagePeersHistoryMutex.Unlock()
			return true
		}
	}

	messagePeersHistoryMutex.Unlock()
	return false
}

func processQueueCommandMessage(message []byte) {
	commandMessage := new(CommandMessage)
	err := json.Unmarshal(message, commandMessage)
	if err != nil {
		logError(P2P_LOG, "Cannot parse incoming command message: %s", err)
		return
	}

	switch commandMessage.Type {
	case EXECUTE_PING:
		msg := sendMessageBroadcast(PING, BlockTransfer{}, "", 0, 0)
		processNetworkMessage(msg)
		break
	case EXECUTE_GENESIS_BLOCK_BUILD:
		// Send node's public key in order to create initial transaction in genesis block for this node from coinbase
		if state != AWAITING_GENESIS_BLOCK {
			//logError(P2P_LOG, "Received message for (%s) state while in the (%s) state", AWAITING_GENESIS_BLOCK, state)
			return
		}

		// Broadcast message so other nodes also start to broadcast their public key for initial transaction
		msg := sendMessageBroadcast(GENESIS_BLOCK_BUILD, BlockTransfer{}, "", 0, 0)
		processNetworkMessage(msg)
		break
	case RETRIEVE_GENESIS_BLOCK:
		if state != AWAITING_GENESIS_BLOCK {
			//logError(P2P_LOG, "Received message for (%s) state while in the (%s) state", AWAITING_GENESIS_BLOCK, state)
			return
		}

		block := commandMessage.GenesisBlock

		// For this particular message type, do not broadcast it further
		processedBlock := processGenesisBlock(block)
		newBlockTransfer := BlockTransfer{
			Hash:                    processedBlock.Hash,
			Number:                  processedBlock.Number,
			CreatedAt:               processedBlock.CreatedAt,
			Transactions:            processedBlock.Transactions,
			MerkleRoot:              processedBlock.MerkleRoot,
			Author:                  processedBlock.Author,
			PreviousBlockHash:       processedBlock.PreviousBlockHash,
			PreviousSecondBlockHash: processedBlock.PreviousSecondBlockHash,
			Depth:                   processedBlock.Depth,
			Row:                     processedBlock.Row,
			Col:                     processedBlock.Col,
			CommitmentHash:          processedBlock.CommitmentHash,
			CommitmentSecret:        processedBlock.CommitmentSecret,
		}

		rmqMessage := RMQMessageBlockBuild{
			Type:   BCHAIN_BLOCK,
			NodeId: selfIdStr,
			Block:  newBlockTransfer,
		}
		sendRabbitMQMessage(rmqMessage)

		// Change state to awaiting blockchain start
		state = AWAITING_BLOCKCHAIN_START
		break
	case START_BLOCKCHAIN:
		if state != AWAITING_BLOCKCHAIN_START {
			logError(P2P_LOG, "Received message for (%s) state while in the (%s) state", AWAITING_BLOCKCHAIN_START, state)
			return
		}

		// Commitment will be broadcast during processing
		msg := sendMessageBroadcast(BCHAIN_INIT_COMMITMENT, BlockTransfer{}, "", 0, 0)
		processNetworkMessage(msg)
		break
	case REQUEST_PUBLIC_KEY_DISTRIBUTION:
		if len(exchangedPubKey) != 1 {
			logError(P2P_LOG, "Public key distribution already started")
		} else {
			exchangedPubKeySentInitialMessage = true
			sendMessageBroadcast(PUB_KEY_EXCHANGE, BlockTransfer{}, getPublicKey(), 0, 0)
			break
		}
	case GATHER_TRANSACTION_PACK:
		var txs []*Transaction
		for _, tx := range commandMessage.Transactions {
			txCopy := Transaction{
				Hash:      tx.Hash,
				HashMask:  0,
				CreatedAt: tx.CreatedAt,
				Recipient: tx.Recipient,
				Sender:    tx.Sender,
				Amount:    tx.Amount,
			}

			txs = append(txs, &txCopy)
		}
		calculateTxHashMasks(txs)

		mempoolMutex.Lock()
		for _, tx := range txs {
			mempool[tx.Hash] = tx
		}
		mempoolMutex.Unlock()

		logInfoBold(P2P_LOG, "Received %d transactions. Current size of mempool (%d)", len(txs), len(mempool))
		break
	case STOP_BLOCKCHAIN:
		if state == AWAITING_GENESIS_BLOCK {
			logError(P2P_LOG, "Received message to stop blockchain while in the (%s) state", AWAITING_BLOCKCHAIN_START, state)
			break
		} else {
			stopNode()
			break
		}
	default:
		logError(P2P_LOG, "Unknown command message type: %d", commandMessage.Type)
		break
	}
}

func initializeBucketSort() {
	sort.Strings(commitmentsBucket)
}

func randomBucketShuffle(fromInit bool) {
	var blockHashBytes []byte
	if !fromInit {
		// Convert strings into bytes
		blockHashBytes, err = hex.DecodeString(RANDOM_SHUFFLE_HEX)
		//logInfoBold(P2P_LOG, "Using hash for randomBucketShuffle: %s", blockHash)

		if err != nil {
			logError(P2P_LOG, "Cannot decode filtered blocks hashes: %s", err)
			return
		}

		logInfoBold(P2P_LOG, "Size of commitmentsBucket: %d", len(commitmentsBucket))
		if len(commitmentsBucket) > 3 {
			logInfoBold(P2P_LOG, "Parts of commitmentsBucket: %s, %s, %s, %s", commitmentsBucket[0][:8], commitmentsBucket[1][:8], commitmentsBucket[2][:8], commitmentsBucket[3][:8])
		}

		if len(commitmentsBucket) != MAX_COMMITMENT_VAR*numOfNodes {
			return
		}
	} else {
		// Genesis block case
		blockHashBytes, err = hex.DecodeString(blockchain[len(blockchain)-1].Hash)

		if err != nil {
			logError(P2P_LOG, "Cannot decode genesis block hash: %s", err)
			return
		}
	}

	// ** Deterministically shuffle commitmentBucket **
	// 1. Convert block hash into a 64bit integer seed using FNV-1a hash algorithm
	hashFnv := fnv.New64a()

	_, err = hashFnv.Write(blockHashBytes)
	if err != nil {
		logError(P2P_LOG, "Cannot initialize FNV-1a hash algorithm: %s", err)
		return
	}

	// 2. Set RNG generator with seed from the genesis block
	seed := int64(hashFnv.Sum64())
	rng := mathRand.New(mathRand.NewSource(seed))

	// 3. Sort commitmentBucket
	initializeBucketSort()

	// 4. Shuffle commitmentBucket
	rng.Shuffle(len(commitmentsBucket), func(i, j int) {
		commitmentsBucket[i], commitmentsBucket[j] = commitmentsBucket[j], commitmentsBucket[i]
	})
}

func identifyCommitmentPositions() (string, []int) {
	// ** Identify node's commitment position **

	var positions []int

	submittedCommitmentsMutex.Lock()
	for i, sharedCommitmentHash := range commitmentsBucket {
		for _, nodeCommitment := range submittedCommitments {
			if sharedCommitmentHash == nodeCommitment.Hash {
				positions = append(positions, i)
				break
			}
		}
	}
	submittedCommitmentsMutex.Unlock()

	logInfo(P2P_LOG, "Positions for epoch (%d): %v", commitmentGenEpochRound, positions)

	// ** Bucket hash calculation **
	// Calculate hash of the bucket from concatenated commitments
	// This is used to compare with other to achieve consistent state
	bucketHash := hex.EncodeToString(keccak256.New().Hash([]byte(strings.Join(commitmentsBucket, ""))))

	return bucketHash, positions
}

func passToEpochReady() {
	// Pass current commitment vars to epoch-ready vars
	rdySubmittedCommitments = make([]CommitmentInfo, len(submittedCommitments))
	copy(rdySubmittedCommitments, submittedCommitments)

	rdyCommitmentsBucket = make([]string, len(commitmentsBucket))
	copy(rdyCommitmentsBucket, commitmentsBucket)

	rdyBucketPositions = make([]int, len(bucketPositions))
	copy(rdyBucketPositions, bucketPositions)

	maps.Copy(rdyCommitmentsBucketMap, commitmentsBucketMap)
	rdyBucketHash = bucketHash
	rdyEpochRound = commitmentGenEpochRound

	commitmentRoundMutex.Lock()
	commitmentGenEpochRound++
	commitmentRoundMutex.Unlock()

	bucketHashesCheckMutex.Lock()
	bucketHashesCheck = []string{}
	bucketHashesCheckMutex.Unlock()

	// Reset current commitment vars
	commitmentBucketMutex.Lock()
	submittedCommitments = []CommitmentInfo{}
	commitmentsBucket = []string{}
	commitmentsBucketMap = make(map[string]CommitmentSharedInfo)
	bucketHash = ""
	bucketPositions = []int{}
	commitmentBucketMutex.Unlock()
}

func passToEpochRunning() {
	// Pass current commitment vars to epoch-running vars
	runSubmittedCommitments = make([]CommitmentInfo, len(rdySubmittedCommitments))
	copy(runSubmittedCommitments, rdySubmittedCommitments)

	runCommitmentsBucket = make([]string, len(rdyCommitmentsBucket))
	copy(runCommitmentsBucket, rdyCommitmentsBucket)

	runBucketPositions = make([]int, len(rdyBucketPositions))
	copy(runBucketPositions, rdyBucketPositions)

	maps.Copy(runCommitmentsBucketMap, rdyCommitmentsBucketMap)
	runBucketHash = rdyBucketHash
	runEpochRound = rdyEpochRound

	// Reset current commitment vars
	rdySubmittedCommitments = []CommitmentInfo{}
	rdyCommitmentsBucket = []string{}
	rdyCommitmentsBucketMap = make(map[string]CommitmentSharedInfo)
	rdyBucketHash = ""
	rdyBucketPositions = []int{}
}

func sendRabbitMqReadyMessage() {
	sendReadyMessageMutex.Lock()
	if sendReadyMessageState {
		sendReadyMessageMutex.Unlock()
		return
	} else {
		sendReadyMessageState = true
		sendReadyMessageMutex.Unlock()

		// Send a RabbitMQ message to the bootstrap server that the node is ready
		readyMessage := RMQMessageNodeReady{
			Type:   NODE_READY,
			NodeId: selfIdStr,
		}
		sendRabbitMQMessage(readyMessage)
		logInfo(P2P_SETUP_LOG, "Node is ready")
	}
}

func generateJitter() int64 {
	// Generate jitter in milliseconds using exponential distribution
	randomValue := randomJitterGenerator.ExpFloat64()
	return int64(math.Round(randomValue * 1_000_000))
}

func getCommitmentInfoByHash(hash string) CommitmentInfo {
	for _, commitmentInfo := range runSubmittedCommitments {
		if commitmentInfo.Hash == hash {
			return commitmentInfo
		}
	}

	logError(P2P_LOG, "Commitment for block to create not found for commitment hash: %s", hash)
	return CommitmentInfo{}
}

func getIdForBufioStream(writer *bufio.ReadWriter) string {
	for peerId, stream := range nodeStreams.Streams {
		if stream.Buffer == writer {
			return peerId
		}
	}
	logFatal(P2P_LOG, "Bufio writer not found in nodeStream structure")
	return ""
}

func createStream(ctx context.Context, node host.Host, streamHandler network.StreamHandler, peerId string, delay float64) {
	// Set a function as stream handler.
	// This function is called when a peer connects, and starts a stream with this protocol.
	// Only applies on the receiving side.
	var builder strings.Builder
	builder.WriteString("/connection/")
	builder.WriteString(ctx.Value("selfIdStr").(string))
	builder.WriteString("_")
	builder.WriteString(peerId)

	streamWrapper := new(StreamWrapper)
	streamWrapper.Stream = nil
	streamWrapper.Buffer = nil
	streamWrapper.Latency = int64(math.Round(delay * 1_000_000)) // Multiply to avoid float precision issues
	nodeStreams.Streams[peerId] = *streamWrapper

	node.SetStreamHandler(protocol.ID(builder.String()), streamHandler)
}

func connectStream(ctx context.Context, h host.Host, destination string, peerId string) (network.Stream, *bufio.ReadWriter, error) {
	// Turn the destination into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(destination)
	if err != nil {
		logError(P2P_LOG, "Failed to create multiaddress for destination: %s", err)
		return nil, nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		logError(P2P_LOG, "Failed to extract peer ID from multiaddress: %s", err)
		return nil, nil, err
	}

	// Add the destination's peer multiaddress in the peerstore.
	// This will be used during connection and stream creation by libp2p.
	h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// Start a stream with the destination.
	// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
	var builder strings.Builder
	builder.WriteString("/connection/")
	builder.WriteString(peerId)
	builder.WriteString("_") // separate peerId from host id
	builder.WriteString(ctx.Value("selfIdStr").(string))

	stream, err := h.NewStream(context.Background(), info.ID, protocol.ID(builder.String()))
	if err != nil {
		logError(P2P_LOG, "Failed to create stream to %s: %s", peerId, err)
		return nil, nil, err
	}

	logInfo(P2P_LOG, "Established connection to %s", peerId)

	// Create a buffered stream so that read and writes are non-blocking.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	return stream, rw, nil
}
