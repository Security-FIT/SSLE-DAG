// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

const PROTOCOL_VERSION = "1"
const ZKP_COMMITMENT_SEPARATOR = "."

const ZKP_COMMITMENT_EPOCH_SEPARATOR = "_"
const ZKP_COMMITMENT_VARIANT_SEPARATOR = "#"

const VOLUME_ENV = "VOLUME_PATH"
const WS_HOSTNAME_ENV = "WS_HOSTNAME_ENV"
const RABBITMQ_HOSTNAME_ENV = "RABBITMQ_HOSTNAME_ENV"

const MAX_MESSAGE_HISTORY_SIZE = 4000
const MAX_EPOCH_ROUNDS_SIZE = 60

const COINBASE_ADDRESS = "0000000000000000000000000000000000000000"

var MIN_COMMITMENT_VAR int // 1
var MAX_COMMITMENT_VAR int // 32

var BLOCK_TIME_SECONDS int // 20
var DAG_PARALLEL_LEN int   // 8 // Must be power of 2
var EPOCH_SIZE_ROUNDS int  // 12
var EPOCH_SIZE int         // DAG_PARALLEL_LEN * EPOCH_SIZE_ROUNDS

var PROCESSED_TX_PER_BLOCK int // 12
var MIN_TX_TO_SPLIT int        // 10
var MIN_TX_TO_MERGE int        // 6

var RANDOM_SHUFFLE_HEX string // "83be9666f14859e7df443a6a4e6ec1c84c9b67a7b3fc2603a4c0f0437164ecf2be"

const PUB_KEY_MERKLE_TREE_SIZE = 512
const PUB_KEY_MERKLE_TREE_DEPTH = 9
