// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"math"
	"math/big"
	mathRand "math/rand"
	"strconv"
	"sync"
	"time"
)

var configPeers []PeerInfo
var nodeStreams NodeStreams
var state State
var expectedNumberOfPeers int

func runP2p(node host.Host, config NodePeerStruct) {
	selfId, err = strconv.Atoi(config.PeerLocation.Id)
	selfIdStr = config.PeerLocation.Id

	if err != nil {
		logFatal(P2P_SETUP_LOG, "Failed to convert Peer ID to integer: %s", err)
	}

	// Create context
	ctx := context.Background()
	ctx = context.WithValue(ctx, "selfId", selfId)
	ctx = context.WithValue(ctx, "selfIdStr", config.PeerLocation.Id)

	expectedNumberOfPeers = len(config.Peers)

	nodeStreams = NodeStreams{Streams: make(map[string]StreamWrapper)}
	messagePeersHistory = make(map[string][]string)
	configPeers = config.Peers

	var streamsSync sync.Mutex
	var wg sync.WaitGroup
	for _, peerInfo := range config.Peers {
		wg.Add(1)
		go func(peerInfo PeerInfo) {
			defer wg.Done()
			peerId, err := strconv.Atoi(peerInfo.Id)
			if err != nil {
				logFatal(P2P_SETUP_LOG, "Failed to convert Peer ID to integer: %s", err)
			}
			if selfId > peerId {
				// A node with greater ID initiates the connection
				streamsSync.Lock()
				createStream(ctx, node, handleStream, peerInfo.Id, peerInfo.Delay)
				streamsSync.Unlock()
			} else {
				time.Sleep(time.Duration(selfId*10)*time.Millisecond + 2000*time.Millisecond)
				var connectRw *bufio.ReadWriter
				var stream network.Stream
				var connectionError error = nil
				var connectionErrorCounter = 0

				streamsSync.Lock()
				stream, connectRw, connectionError = connectStream(ctx, node, peerInfo.P2PId, peerInfo.Id)
				streamsSync.Unlock()
				for connectionError != nil && connectionErrorCounter < 2048 {
					time.Sleep(5000 * time.Millisecond)
					streamsSync.Lock()
					stream, connectRw, connectionError = connectStream(ctx, node, peerInfo.P2PId, peerInfo.Id)
					streamsSync.Unlock()

					if connectionError != nil {
						logError(P2P_SETUP_LOG, "Failed to connect to peer %s, retrying in 5000ms...", peerInfo.Id)
						time.Sleep(1000 * time.Millisecond)
						connectionErrorCounter++
						continue
					}
				}

				if connectionError != nil {
					logFatal(P2P_SETUP_LOG, "Failed to connect to peer %s", peerInfo.Id)
				}

				// Create a thread to read and write data.
				go readData(connectRw)

				streamWrapper := new(StreamWrapper)
				streamWrapper.Buffer = connectRw
				streamWrapper.Stream = stream
				streamWrapper.Latency = int64(math.Round(peerInfo.Delay * 1_000_000))
				nodeStreams.Streams[peerInfo.Id] = *streamWrapper

				if len(nodeStreams.Streams) == expectedNumberOfPeers {
					sendRabbitMqReadyMessage()
				}
			}
		}(peerInfo)
	}
	wg.Wait()

	logInfo(P2P_SETUP_LOG, "All peers connected")

	// Initialize variables
	commitmentGenEpochRound = 0
	bucketInfoStarted = false
	commitmentsBucketMap = make(map[string]CommitmentSharedInfo)
	rdyCommitmentsBucketMap = make(map[string]CommitmentSharedInfo)
	runCommitmentsBucketMap = make(map[string]CommitmentSharedInfo)
	timeSynced = false
	mempool = make(map[string]*Transaction)

	blockchainLength = 0
	slot = 0
	absRound = 0

	exchangedPubKey = append(exchangedPubKey, getPublicKey())
	exchangedPubKeySentInitialMessage = false
	leafHashes = make([][]byte, PUB_KEY_MERKLE_TREE_SIZE)

	state = AWAITING_GENESIS_BLOCK

	randomJitterGenerator = mathRand.New(mathRand.NewSource(time.Now().UnixNano()))

	maskConstant, successfulMaskConversion = new(big.Int).SetString("80", 16)
	if !successfulMaskConversion {
		logFatal(P2P_SETUP_LOG, "Failed to convert mask cont to big integer")
	}
}

func initP2p() (host.Host, multiaddr.Multiaddr) {
	// start a libp2p node that listens on a random local TCP port,
	// but without running the built-in ping protocol
	privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)

	if err != nil {
		panic(err)
	}

	node, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Identity(privKey),
	)

	if err != nil {
		panic(err)
	}

	peerInfo := peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}

	return node, addrs[1]
}

func initNode() {
	sendReadyMessageState = false
}
