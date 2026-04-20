// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark-crypto/signature"
	eddsa2 "github.com/consensys/gnark-crypto/signature/eddsa"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	twistededwards2 "github.com/consensys/gnark/std/algebra/native/twistededwards"
	"github.com/consensys/gnark/std/hash/mimc"
	eddsa3 "github.com/consensys/gnark/std/signature/eddsa"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var pk groth16.ProvingKey
var vk groth16.VerifyingKey

var publicKey string
var privateKey signature.Signer

var msgLength int

type EddsaCircuit struct {
	PublicKey       eddsa3.PublicKey  `gnark:",secret"`
	Signature       eddsa3.Signature  `gnark:",public"`
	Message         frontend.Variable `gnark:",public"`
	Secret          frontend.Variable `gnark:",secret"`
	MessageSummed   frontend.Variable `gnark:",secret"`
	SecretSignature eddsa3.Signature  `gnark:",secret"`
	EpochRound      frontend.Variable `gnark:",public"`

	Root        frontend.Variable                            `gnark:",public"` // public: Merkle root
	PathNodes   [PUB_KEY_MERKLE_TREE_DEPTH]frontend.Variable `gnark:",secret"` // public: sibling nodes
	PathIndices [PUB_KEY_MERKLE_TREE_DEPTH]frontend.Variable `gnark:",secret"` // public: index bits (0 = left, 1 = right)
	Leaf        frontend.Variable                            `gnark:",secret"` // secret: leaf hash
}

type EddsaPreparation struct {
	PublicKey       []byte
	PrivateKey      []byte
	Message         []byte
	Signature       []byte
	Secret          []byte
	MessageSummed   []byte
	SecretSignature []byte
	EpochRound      []byte
}

func initZkp() {
	pk, vk = readKeys()

	privateKey = generatePrivateKey()
	publicKey = getPublicKey()

	msgLength = prepareEdDSAMessageLength()
}

func readKeys() (groth16.ProvingKey, groth16.VerifyingKey) {
	var pk groth16.ProvingKey
	var vk groth16.VerifyingKey

	pkPath := filepath.Join(volumePath, "pk.bin")
	pkFile, err := os.Open(pkPath)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to open pk.bin file: %s", err)
	}
	pk = groth16.NewProvingKey(ecc.BLS12_381)
	_, err = pk.ReadFrom(pkFile)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to read from pk.bin file: %s", err)
	}
	err = pkFile.Close()
	if err != nil {
		logFatal(ZKP_LOG, "Failed to close pk.bin file: %s", err)
	}

	vkPath := filepath.Join(volumePath, "vk.bin")
	vkFile, err := os.Open(vkPath)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to open vk.bin file: %s", err)
	}
	vk = groth16.NewVerifyingKey(ecc.BLS12_381)
	_, err = vk.ReadFrom(vkFile)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to read from vk.bin file: %s", err)
	}
	err = vkFile.Close()
	if err != nil {
		logFatal(ZKP_LOG, "Failed to close vk.bin file: %s", err)
	}

	return pk, vk
}

func generateCommitment(epoch uint32, variant uint64) (witness.Witness, groth16.Proof, string, string) {
	var circuit EddsaCircuit
	ccs, err := frontend.Compile(ecc.BLS12_381.ScalarField(), r1cs.NewBuilder, &circuit)

	if err != nil {
		logFatal(ZKP_LOG, "Failed to compile circuit: %s", err)
	}

	preparation := prepareEdDSA(privateKey, epoch, variant)

	var assignment EddsaCircuit
	(*eddsa3.PublicKey).Assign(&assignment.PublicKey, twistededwards.BLS12_381, preparation.PublicKey)
	(*eddsa3.Signature).Assign(&assignment.Signature, twistededwards.BLS12_381, preparation.Signature)
	assignment.Message = preparation.Message
	assignment.Secret = preparation.Secret
	assignment.MessageSummed = preparation.MessageSummed
	assignment.EpochRound = preparation.EpochRound
	(*eddsa3.Signature).Assign(&assignment.SecretSignature, twistededwards.BLS12_381, preparation.SecretSignature)
	assignment.Root = pubMerkleTreeRoot
	assignment.Leaf = leafHashes[pubKeyLeafIndex]

	for i := 0; i < PUB_KEY_MERKLE_TREE_DEPTH; i++ {
		assignment.PathNodes[i] = pathNodes[i]
		assignment.PathIndices[i] = pathIndices[i]
	}

	newWitness, err := frontend.NewWitness(&assignment, ecc.BLS12_381.ScalarField())
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create new witness: %s", err)
	}

	publicWitness, err := newWitness.Public()
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create public witness: %s", err)
	}

	proof, err := groth16.Prove(ccs, pk, newWitness)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to generate proof for commitment: %s", err)
	}

	return publicWitness, proof, hex.EncodeToString(preparation.Secret), hex.EncodeToString(preparation.SecretSignature)
}

func generatePrivateKey() signature.Signer {
	// Generate private key (A) from randomness (a)
	privKey, err := eddsa2.New(twistededwards.BLS12_381, rand.Reader)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to generate EdDSA key: %s", err)
	}

	return privKey
}

func prepareEdDSA(privateKey signature.Signer, epoch uint32, variant uint64) EddsaPreparation {
	epochStr := fmt.Sprintf("%010d", epoch)
	variantStr := fmt.Sprintf("%05d", variant)

	// Message require version of protocol and consensus epoch separated by ZKP_COMMITMENT_EPOCH_SEPARATOR char and variant separated by ZKP_COMMITMENT_VARIANT_SEPARATOR char
	var messageBuilder strings.Builder
	messageBuilder.WriteString(PROTOCOL_VERSION)
	messageBuilder.WriteString(ZKP_COMMITMENT_EPOCH_SEPARATOR)
	messageBuilder.WriteString(epochStr)
	messageBuilder.WriteString(ZKP_COMMITMENT_VARIANT_SEPARATOR)
	messageBuilder.WriteString(variantStr)

	epochBytes := []byte(epochStr)

	// public key is used as blockchain user identification and all funds are attached to it
	pubKey := privateKey.Public()

	// ** Secret signature **
	// Create secret signature
	sigSecret, err := privateKey.Sign(epochBytes, hash.Hash.New(hash.MIMC_BLS12_381))
	if err != nil {
		logFatal(ZKP_LOG, "Failed to sign secret signature: %s", err)
	}

	// Verify secret signature
	checkVerifySecret, err := pubKey.Verify(sigSecret, epochBytes, hash.Hash.New(hash.MIMC_BLS12_381))
	if err != nil || !checkVerifySecret {
		logFatal(ZKP_LOG, "Failed to verify secret signature: %s", err)
	}

	// Take only the first 31 bytes of secret because of the undefined behavior
	// of the circuit for input of size 32 bytes or more
	secret := sigSecret[:31]

	msg := []byte(messageBuilder.String())

	// Combine message and secret to create a unique hash which in message signature
	hasherSecret := hash.MIMC_BLS12_381.New()
	hasherSecret.Reset()
	hasherSecret.Write(msg)
	hasherSecret.Write(secret)
	messageSummed := hasherSecret.Sum(nil)

	// ** Message signature **
	// Create message signature
	sigMsg, err := privateKey.Sign(messageSummed, hash.Hash.New(hash.MIMC_BLS12_381))
	if err != nil {
		logFatal(ZKP_LOG, "Failed to sign message signature: %s", err)
	}

	// Verify message signature
	checkVerifyMessage, err := pubKey.Verify(sigMsg, messageSummed, hash.Hash.New(hash.MIMC_BLS12_381))
	if err != nil || !checkVerifyMessage {
		logFatal(ZKP_LOG, "Failed to verify message signature: %s", err)
	}

	return EddsaPreparation{
		PublicKey:       pubKey.Bytes(),
		PrivateKey:      privateKey.Bytes(),
		Message:         msg,
		Signature:       sigMsg,
		Secret:          secret,
		MessageSummed:   messageSummed,
		SecretSignature: sigSecret,
		EpochRound:      epochBytes,
	}
}

func prepareEdDSAMessageLength() int {
	return len(PROTOCOL_VERSION) + len(ZKP_COMMITMENT_EPOCH_SEPARATOR) + 10 + len(ZKP_COMMITMENT_VARIANT_SEPARATOR) + 5
}

func commitmentBase64Encode(publicWitness witness.Witness, proof groth16.Proof) string {
	var bufWitness bytes.Buffer
	_, err = publicWitness.WriteTo(&bufWitness)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to write public witness to buffer: %s", err)
	}
	publicWitnessBinary := bufWitness.Bytes()

	var bufProof bytes.Buffer
	_, err = proof.WriteTo(&bufProof)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to write proof to the buffer: %s", err)
	}
	proofBinary := bufProof.Bytes()

	var commitmentBuilder strings.Builder
	commitmentBuilder.WriteString(base64.StdEncoding.EncodeToString(publicWitnessBinary))
	commitmentBuilder.WriteString(ZKP_COMMITMENT_SEPARATOR)
	commitmentBuilder.WriteString(base64.StdEncoding.EncodeToString(proofBinary))

	return commitmentBuilder.String()
}

func commitmentBase64Decode(data string) (witness.Witness, groth16.Proof, bool) {
	splits := strings.Split(data, ZKP_COMMITMENT_SEPARATOR)
	if len(splits) != 2 {
		logError(ZKP_LOG, "Invalid commitment format (%d) splits in (%s)", len(splits), data)
		return nil, nil, false
	}

	publicWitnessBinary, err := base64.StdEncoding.DecodeString(splits[0])
	if err != nil {
		logError(ZKP_LOG, "Failed to decode publicWitness: %s", err)
		return nil, nil, false
	}

	publicWitness, err := witness.New(ecc.BLS12_381.ScalarField())
	if err != nil {
		logError(ZKP_LOG, "Failed to create new witness: %s", err)
		return nil, nil, false
	}
	_, err = publicWitness.ReadFrom(bytes.NewReader(publicWitnessBinary))
	if err != nil {
		logError(ZKP_LOG, "Failed to unmarshal publicWitness: %s", err)
		return nil, nil, false
	}

	proofBinary, err := base64.StdEncoding.DecodeString(splits[1])
	if err != nil {
		logError(ZKP_LOG, "Failed to decode proof: %s", err)
		return nil, nil, false
	}

	var proof groth16.Proof
	proof = groth16.NewProof(ecc.BLS12_381)
	_, err = proof.ReadFrom(bytes.NewReader(proofBinary))
	if err != nil {
		logError(ZKP_LOG, "Failed to unmarshal proof: %s", err)
		return nil, nil, false
	}

	return publicWitness, proof, true
}

func decodePublicWitness(publicWitness witness.Witness) (PublicWitness, bool) {
	var bufWitness bytes.Buffer
	_, err = publicWitness.WriteTo(&bufWitness)
	if err != nil {
		logError(ZKP_LOG, "Failed to deconstruct public witness bytes to buffer: %s", err)
		return PublicWitness{}, false
	}
	publicWitnessHex := hex.EncodeToString(bufWitness.Bytes())

	// Skip first 24 bytes reserved for a number of variables
	// Note: 3 for message signature (Rx, Ry, S)
	const START_INDEX = 24

	// Decode hex form into utf-8
	utf8Message, err := hex.DecodeString(publicWitnessHex[START_INDEX+64*3 : START_INDEX+64*4])

	if err != nil {
		logError(ZKP_LOG, "Failed to decode hex string for Message in public witness: %s", err)
		return PublicWitness{}, false
	}

	// Take only last N chars of message as other is only 0 padding
	utf8MessageString := string(utf8Message[len(string(utf8Message))-msgLength:])

	decodedPublicWitness := PublicWitness{
		MessageSigRX: publicWitnessHex[START_INDEX+64*0 : START_INDEX+64*1],
		MessageSigRY: publicWitnessHex[START_INDEX+64*1 : START_INDEX+64*2],
		MessageSigS:  publicWitnessHex[START_INDEX+64*2 : START_INDEX+64*3],
		Message:      utf8MessageString,
		EpochRound:   publicWitnessHex[START_INDEX+64*4 : START_INDEX+64*5],
	}

	return decodedPublicWitness, true
}

func isCommitmentValid(commitment string, epoch uint32) (bool, PublicWitness) {
	// - validate JSON message
	// - validate content separator between base64 public witness and base64 proof
	// - validate correct base64 of public witness
	// - validate correct base64 of proof
	// - validate correct content of public witness
	// - validate separator in public witness's message
	// - validate correct protocol version
	// - validate correct epoch
	// - validate correct variant number
	// - validate ZKP proof

	// Public is unique for each node but same for each commitment (hidden)
	// Because witness's message is signed by private key and can be verified by signature it cannot be counterfeited
	// This way, a node cannot spread multiple commitment per account
	// It should be noted that initial stake like 32ETH is important by using this approach as:
	// 		- nodes need to have some locked value in case of adversary behaviour
	// 		- create more stable opportunity to attend election and become leader as it costs some proportion part of tokens
	publicWitness, proof, success := commitmentBase64Decode(commitment)
	if !success {
		return false, PublicWitness{}
	}

	decodedPublicWitness, success := decodePublicWitness(publicWitness)
	if !success {
		return false, PublicWitness{}
	}

	splitEpoch := strings.Split(decodedPublicWitness.Message, ZKP_COMMITMENT_EPOCH_SEPARATOR)
	protocol := splitEpoch[0]

	if protocol != PROTOCOL_VERSION {
		logError(ZKP_LOG, "Invalid protocol version (%s), expected (%s)", protocol, PROTOCOL_VERSION)
		return false, PublicWitness{}
	}

	splitVariant := strings.Split(splitEpoch[1], ZKP_COMMITMENT_VARIANT_SEPARATOR)
	epochRound, err := strconv.ParseUint(splitVariant[0], 10, 64)
	if err != nil {
		logError(ZKP_LOG, "Cannot parse epoch round (%s)", splitVariant[0])
		return false, PublicWitness{}
	}

	if uint32(epochRound) != epoch {
		logError(ZKP_LOG, "Invalid epoch round (%d), expected (%d)", epochRound, epoch)
		return false, PublicWitness{}
	}

	decodedEpochBytes, err := hex.DecodeString(decodedPublicWitness.EpochRound)
	if err != nil {
		logError(ZKP_LOG, "Failed to decode hex string for EpochRound in public witness: %s", err)
		return false, PublicWitness{}
	}
	decodedEpochString := string(decodedEpochBytes[len(string(decodedEpochBytes))-10:])
	convertedEpochRound, err := strconv.ParseUint(decodedEpochString, 10, 32)
	if err != nil {
		logError(ZKP_LOG, "Failed to convert epoch round (%s) to uint32: %s", decodedPublicWitness.EpochRound, err)
		return false, PublicWitness{}
	}
	if uint32(convertedEpochRound) != epoch {
		logError(ZKP_LOG, "Invalid epoch round (%d) as public variable (%s), expected (%d)", convertedEpochRound, decodedPublicWitness.EpochRound, epoch)
		return false, PublicWitness{}
	}

	variantNumber, err := strconv.ParseUint(splitVariant[1], 10, 64)
	if err != nil {
		logError(ZKP_LOG, "Cannot parse variant number (%s)", splitVariant[1])
		return false, PublicWitness{}
	}

	if (variantNumber < uint64(MIN_COMMITMENT_VAR)) || (variantNumber > uint64(MAX_COMMITMENT_VAR)) {
		logError(ZKP_LOG, "Invalid variant number (%d), expected between (%d) and (%d) (including both)", variantNumber, MIN_COMMITMENT_VAR, MAX_COMMITMENT_VAR)
		return false, PublicWitness{}
	}

	// Verify ZKP proof
	err = groth16.Verify(proof, vk, publicWitness)
	if err != nil {
		logError(ZKP_LOG, "Failed to verify ZKP proof: %s", err)
		return false, PublicWitness{}
	}

	return true, decodedPublicWitness
}

func (circuit *EddsaCircuit) Define(api frontend.API) error {
	curve, err := twistededwards2.NewEdCurve(api, twistededwards.BLS12_381)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create Twisted Edwards curve: %s", err)
	}

	hFuncSum, err := mimc.NewMiMC(api)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create MiMC hash function: %s", err)
	}
	hFuncSum.Reset()
	hFuncSum.Write(circuit.Message)
	hFuncSum.Write(circuit.Secret)

	// Check if the provided combined message hash is equal to the hash of the message and secret
	api.AssertIsEqual(circuit.MessageSummed, hFuncSum.Sum())

	hFunc, err := mimc.NewMiMC(api)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create MiMC hash function: %s", err)
	}
	hFuncSecret, err := mimc.NewMiMC(api)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to create MiMC hash function: %s", err)
	}

	// Verify signature used for secret
	err = eddsa3.Verify(curve, circuit.SecretSignature, circuit.EpochRound, circuit.PublicKey, &hFuncSecret)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to verify EdDSA signature using MiMC hash for secret: %s", err)
	}

	// Verify signature used for message (commitment data)
	err = eddsa3.Verify(curve, circuit.Signature, circuit.MessageSummed, circuit.PublicKey, &hFunc)
	if err != nil {
		logFatal(ZKP_LOG, "Failed to verify EdDSA signature using MiMC hash for message: %s", err)
	}

	// Create merkle proof with connection to a public key
	hFunc.Reset()

	currentNode := circuit.Leaf

	// Iterate the tree
	for i := 0; i < PUB_KEY_MERKLE_TREE_DEPTH; i++ {
		// Select the order depending on the index bit
		left := api.Select(api.IsZero(circuit.PathIndices[i]), currentNode, circuit.PathNodes[i])
		right := api.Select(api.IsZero(circuit.PathIndices[i]), circuit.PathNodes[i], currentNode)

		// Hash the pair
		hFunc.Reset()
		hFunc.Write(left)
		hFunc.Write(right)
		currentNode = hFunc.Sum()
	}

	// Verify computed root must equal provided root
	api.AssertIsEqual(currentNode, circuit.Root)

	// Additionally, it needs to be checked if the hash of the leaf is equal to the hash of a public key provided in circuit
	// For the current implementation, it was omitted as the current implementation of a gnark library (1st May 2023)
	// contains a bug which causes to randomly output hash with 0 value for input of size 32 bytes or more and leads
	// to undefined behavior of the circuit

	api.Println(circuit.PublicKey)
	api.Println(circuit.Signature)
	api.Println(circuit.Message)
	api.Println(circuit.Secret)
	api.Println(circuit.MessageSummed)
	api.Println(circuit.SecretSignature)
	api.Println(circuit.Root)

	return nil
}

func getPublicKey() string {
	// return hex.EncodeToString(keccak256.New().Hash(privateKey.Public().Bytes()))
	return hex.EncodeToString(privateKey.Public().Bytes())
}
