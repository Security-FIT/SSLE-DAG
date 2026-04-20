// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

type State string

const (
	AWAITING_GENESIS_BLOCK      State = "AWAITING_GENESIS_BLOCK"
	AWAITING_BLOCKCHAIN_START   State = "AWAITING_BLOCKCHAIN_START"
	RECEIVING_INIT_COMMITMENTS  State = "RECEIVING_COMMITMENTS"
	VALIDATING_INIT_COMMITMENTS State = "VALIDATING_COMMITMENTS"
	BLOCKCHAIN_RUNNING          State = "RUNNING"
)

type NetworkMessage struct {
	Id                   string        `json:"id"`   // Nonce + Unix timestamp in microseconds
	Type                 Message       `json:"type"` // String identifier of the message
	BlockTransfer        BlockTransfer `json:"blocktransfer"`
	Commitment           string        `json:"commitment"` // Commitment base64 string contains encoded binary data
	CommitmentEpochRound uint32        `json:"commitmentEpochRound"`
	ExtraInt             int64         `json:"extraInt"`
}

type Message int

const (
	BCHAIN_INIT_COMMITMENT Message = iota
	BCHAIN_COMMITMENT
	BCHAIN_INIT_BUCKET_INFO
	BCHAIN_BUCKET_CHECK
	BCHAIN_SYNC_TIME
	BCHAIN_BLOCK
	PING
	GENESIS_BLOCK_BUILD
	INFO
	TRANSACTION_PACK
	PUB_KEY_EXCHANGE
	NODE_READY
)

type CommandMessageType int

const (
	EXECUTE_PING CommandMessageType = iota
	EXECUTE_GENESIS_BLOCK_BUILD
	RETRIEVE_GENESIS_BLOCK
	START_BLOCKCHAIN
	REQUEST_PUBLIC_KEY_DISTRIBUTION
	GATHER_TRANSACTION_PACK
	STOP_BLOCKCHAIN
)

type Block struct {
	Hash                    string        `json:"hash"`
	Number                  uint32        `json:"number"`
	CreatedAt               int64         `json:"createdAt"`
	Transactions            []Transaction `json:"transactions"`
	MerkleRoot              string        `json:"merkleRoot"`
	Author                  string        `json:"author"`
	PreviousBlock           *Block        `json:"previousBlock"`           // for faster access
	PreviousBlockHash       string        `json:"previousBlockHash"`       // mandatory
	PreviousSecondBlock     *Block        `json:"previousSecondBlock"`     // optional (required after merge)
	PreviousSecondBlockHash string        `json:"previousSecondBlockHash"` // optional (for faster access)
	Depth                   int           `json:"depth"`
	Row                     int           `json:"row"`
	Col                     int           `json:"col"`
	CommitmentHash          string        `json:"commitmentHash"`
	CommitmentSecret        string        `json:"commitmentSecret"`
	CommitmentSig           string        `json:"commitmentSig"`
}

type BlockTransfer struct {
	Hash                    string        `json:"hash"`
	Number                  uint32        `json:"number"`
	CreatedAt               int64         `json:"createdAt"`
	Transactions            []Transaction `json:"transactions"`
	MerkleRoot              string        `json:"merkleRoot"`
	Author                  string        `json:"author"`
	PreviousBlockHash       string        `json:"previousBlockHash"`       // mandatory
	PreviousSecondBlockHash string        `json:"previousSecondBlockHash"` // optional (for faster access)
	Depth                   int           `json:"depth"`
	Row                     int           `json:"row"`
	Col                     int           `json:"col"`
	CommitmentHash          string        `json:"commitmentHash"`
	CommitmentSecret        string        `json:"commitmentSecret"`
	CommitmentSig           string        `json:"commitmentSig"`
}

type Transaction struct {
	Hash      string `json:"hash"`
	HashMask  int    `json:"hashMask"` // is empty once tx is received - it's calculated by the node
	CreatedAt int64  `json:"createdAt"`
	Recipient string `json:"recipient"`
	Sender    string `json:"sender"`
	Amount    int64  `json:"amount"`
}

type PublicWitness struct {
	MessageSigRX string
	MessageSigRY string
	MessageSigS  string
	Message      string
	EpochRound   string
}

type GenesisBlock struct {
	Hash         string        `json:"hash"`
	Number       uint32        `json:"number"`
	CreatedAt    int64         `json:"createdAt"`
	Transactions []Transaction `json:"transactions"`
	Author       string        `json:"author"`
}

type GenesisTransaction struct {
	CreatedAt int64  `json:"createdAt"`
	Recipient string `json:"recipient"`
	Sender    string `json:"sender"`
	Amount    int64  `json:"amount"`
}

type CommandMessage struct {
	Type         CommandMessageType `json:"type"`
	Headers      map[string]any     `json:"headers"`
	GenesisBlock GenesisBlock       `json:"genesisBlock"`
	Transactions []Transaction      `json:"transactions"`
}

type CommitmentInfo struct {
	Commitment string
	Hash       string
	Secret     string
	SecretSig  string
}

type CommitmentSharedInfo struct {
	Commitment string
	Hash       string
	Public     PublicWitness
}

type DagStructureAction int

const (
	DAG_ACTION_SPLIT DagStructureAction = iota
	DAG_ACTION_SPLIT_SND
	DAG_ACTION_MERGE
	DAG_ACTION_CONTINUE
	DAG_ACTION_SKIP
)

type DagActionType struct {
	Action DagStructureAction
	Parent *Block
	Second *Block
}

type BlockCreation struct {
	CommitmentHash            string
	CommitmentSecret          string
	CommitmentSecretSignature string
	Transactions              []Transaction
	Depth                     int
	ParentBlock               *Block
	SecondParentBlock         *Block
	Row                       int
	Col                       int
}

type RMQMessagePing struct {
	Id       string  `json:"id"`
	Type     Message `json:"type"`
	NodeId   string  `json:"nodeId"`
	NodeTime int64   `json:"nodeTime"`
}

type RMQMessageGenesisBlockBuild struct {
	Type      Message `json:"type"`
	NodeId    string  `json:"nodeId"`
	PublicKey string  `json:"publicKey"`
}

type RMQMessageBlockBuild struct {
	Type   Message       `json:"type"`
	NodeId string        `json:"nodeId"`
	Block  BlockTransfer `json:"block"`
}

type RMQMessageCommitment struct {
	Type          Message       `json:"type"`
	NodeId        string        `json:"nodeId"`
	Commitment    string        `json:"commitment"`
	PublicWitness PublicWitness `json:"publicWitness"`
	EpochRound    uint32        `json:"epochRound"`
}

type RMQMessageBucketInfo struct {
	Type              Message `json:"type"`
	NodeId            string  `json:"nodeId"`
	BucketLength      int     `json:"bucketLength"`
	BucketHash        string  `json:"bucketHash"`
	SelectedPositions []int   `json:"selectedPositionsLength"`
}

type RMQMessageBucketCheck struct {
	Type   Message `json:"type"`
	NodeId string  `json:"nodeId"`
	Match  float64 `json:"match"`
}

type RMQMessageTimeSync struct {
	Type     Message `json:"type"`
	NodeId   string  `json:"nodeId"`
	SyncTime int64   `json:"syncTime"` // Unix timestamp
}

type RMQMessagePublicKeysSynced struct {
	Type   Message `json:"type"`
	NodeId string  `json:"nodeId"`
}

type RMQMessageNodeReady struct {
	Type   Message `json:"type"`
	NodeId string  `json:"nodeId"`
}

type RMQMessageError struct {
	Type    Message `json:"type"`
	NodeId  string  `json:"nodeId"`
	Message string  `json:"message"`
}
