// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"fmt"
	"github.com/wagslane/go-rabbitmq"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var selfId int
var selfIdStr string
var err error

var volumePath string
var wsHostname string
var rabbitmqHostname string

var rabbitMqConn *rabbitmq.Conn
var rmqCommandsConsumer *rabbitmq.Consumer
var rmqBlockchainPublisher *rabbitmq.Publisher

func main() {
	watchInterrupt()

	if disableDebug {
		log.SetOutput(io.Discard)
	}

	// Read environment variables
	setEnvVariables()

	// Init config
	nodeConfig := readConfig()
	initConfiguration(nodeConfig)

	// Init keys for ZKP
	initZkp()

	// Init p2p node client
	node, addr := initP2p()

	logInfo(MAIN, "Node ID: %s", node.ID())
	logInfo(MAIN, "Full P2P address: %s", addr)

	// Init node
	initNode()

	config := bootstrap(nodeConfig.Bootstrap.Port, addr.String())
	logInfo(MAIN, "Node identity: %s - %s", config.PeerLocation.Id, config.PeerLocation.Name)

	// Init RabbitMQ consumer and publisher connections
	rabbitMqConn = initRabbitMqConnection()
	rmqCommandsConsumer = initRabbitMqConsumers(config.PeerLocation.Id)
	rmqBlockchainPublisher = initRabbitMqPublisher()

	go runRabbitMqCommandConsumer(rmqCommandsConsumer)

	// Init P2P connections
	runP2p(node, *config)

	select {}
}

func watchInterrupt() {
	sigs := make(chan os.Signal, 1)
	errs := make(chan error, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			fmt.Printf("\u001B[2D") // Remove ^C from the output
			logInfo(MAIN, "Received signal: %s", sig.String())
		case err := <-errs:
			logError(MAIN, "Error: %s", err)
		}

		stopNode()
	}()
}

func stopNode() {
	logInfo(MAIN, "Stopping the node...")

	// close Websocket connection
	// close P2P connections
	for _, stream := range nodeStreams.Streams {
		if stream.Stream != nil {
			stream.Stream.Close()
		}
	}

	// close RabbitMQ connection
	if rabbitMqConn != nil {
		rabbitMqConn.Close()
	}
	if rmqCommandsConsumer != nil {
		rmqCommandsConsumer.Close()
	}
	if rmqBlockchainPublisher != nil {
		rmqBlockchainPublisher.Close()
	}

	os.Exit(0)
}
