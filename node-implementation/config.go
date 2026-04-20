// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"github.com/consensys/gnark/logger"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Environment struct {
		Nodes                 int    `yaml:"nodes"`
		BlockTimeSeconds      int    `yaml:"block_time_seconds"`
		DagParallelLength     int    `yaml:"dag_parallel_length"`
		EpochSizeRounds       int    `yaml:"epoch_size_rounds"`
		TxPerBlock            int    `yaml:"tx_per_block"`
		MinTxToSplit          int    `yaml:"min_tx_to_split"`
		MinTxToMerge          int    `yaml:"min_tx_to_merge"`
		RandomShufflingKey    string `yaml:"random_shuffling_key"`
		PubKeyMerkleSize      int    `yaml:"pub_key_merkle_size"`
		PubKeyMerkleTreeDepth int    `yaml:"pub_key_merkle_tree_depth"`
		MinCommitments        int    `yaml:"min_commitments"`
		MaxCommitments        int    `yaml:"max_commitments"`
		SimulateRounds        int    `yaml:"simulate_rounds"`
	}
	Bootstrap struct {
		Port uint16 `yaml:"port"`
	} `yaml:"bootstrap"`
	RabbitMq struct {
		Port uint16 `yaml:"port"`
	} `yaml:"rabbitmq"`
}

var numOfNodes int
var numOfOtherNodes int

var dagMaxDepth int

var disableDebug = true

func setEnvVariables() {
	volumePath = os.Getenv(VOLUME_ENV)
	wsHostname = os.Getenv(WS_HOSTNAME_ENV)
	rabbitmqHostname = os.Getenv(RABBITMQ_HOSTNAME_ENV)

	if volumePath == "" {
		logError(CFG_LOG, "No environment variable %s found, aborting...", VOLUME_ENV)
		os.Exit(1)
	}
}

func readConfig() *Config {
	configFile, err := os.ReadFile(volumePath + "/config.yaml")
	if err != nil {
		logError(CFG_LOG, "Error reading config.yaml: %s", err)
	}

	config := new(Config)
	err = yaml.Unmarshal(configFile, &config)

	if err != nil {
		logError(CFG_LOG, "Error parsing config.yaml: %s", err)
	}

	return config
}

func initConfiguration(config *Config) {
	numOfNodes = config.Environment.Nodes
	numOfOtherNodes = numOfNodes - 1

	MIN_COMMITMENT_VAR = config.Environment.MinCommitments
	MAX_COMMITMENT_VAR = config.Environment.MaxCommitments

	BLOCK_TIME_SECONDS = config.Environment.BlockTimeSeconds
	DAG_PARALLEL_LEN = config.Environment.DagParallelLength
	EPOCH_SIZE_ROUNDS = config.Environment.EpochSizeRounds
	EPOCH_SIZE = DAG_PARALLEL_LEN * EPOCH_SIZE_ROUNDS

	PROCESSED_TX_PER_BLOCK = config.Environment.TxPerBlock
	MIN_TX_TO_SPLIT = config.Environment.MinTxToSplit
	MIN_TX_TO_MERGE = config.Environment.MinTxToMerge

	RANDOM_SHUFFLE_HEX = config.Environment.RandomShufflingKey

	// validate configuration
	if DAG_PARALLEL_LEN&(DAG_PARALLEL_LEN-1) != 0 {
		logError(CFG_LOG, "DAG_PARALLEL_LEN must be a power of 2")
		os.Exit(1)
	}

	dagMaxDepth = logBase2(DAG_PARALLEL_LEN)

	if disableDebug {
		logger.Disable()
	}
}
