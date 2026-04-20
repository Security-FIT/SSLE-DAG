// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type SendIdentityMessage struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type ReceiveTypeMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type ReceiveConfigMessage struct {
	Type string         `json:"type"`
	Data NodePeerStruct `json:"data"`
}

type NodePeerStruct struct {
	PeerLocation PeerLocation `json:"loc"`
	Peers        []PeerInfo   `json:"connections"`
}

type PeerInfo struct {
	Id    string  `json:"id"`
	Delay float64 `json:"delay"`
	P2PId string  `json:"p2p_id"`
}

type PeerLocation struct {
	Id   string `json:"id"`
	Name string `json:"name"` // city
	Lat  string `json:"lat"`
	Lon  string `json:"lon"`
}

type WebsocketState int

const (
	StateSendHello WebsocketState = iota
	StateReceiveHelloConfirm
	StateReceiveConfig
	StateReceiveReady
	StateFinish
)

func bootstrap(port uint16, id string) *NodePeerStruct {
	state := StateSendHello
	flag.Parse()
	//log.SetFlags(0)

	configMessage := new(ReceiveConfigMessage)
	readyToClose := false

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var addr = flag.String("addr", wsHostname+":"+strconv.Itoa(int(port)), "ws service address")
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	logInfo(BTSTRP_LOG, "Connecting to %s ...", u.String())

	connected := false
	var conn *websocket.Conn
	for i := 0; !connected && i < 10; i++ {
		var err error
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			logError(BTSTRP_LOG, "Websocket failed to connect to %s", err)
			logError(BTSTRP_LOG, "Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
		} else {
			connected = true
		}
	}

	if !connected {
		logFatal(BTSTRP_LOG, "Websocket failed to connect, aborting...")
	}

	defer conn.Close()

	done := make(chan struct{})

	// Read received message
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil && !readyToClose {
				logError(BTSTRP_LOG, "Websocket error: %s", err)
				return
			}

			switch state {
			case StateReceiveHelloConfirm:
				helloConfirmation := new(ReceiveTypeMessage)
				err = json.Unmarshal(message, &helloConfirmation)
				if err != nil {
					logError(BTSTRP_LOG, "Unmarshal hello message error: %s", err)
				}

				if helloConfirmation.Data == "ok" {
					logDebug(BTSTRP_LOG, "Received hello confirmation: OK")
					state = StateReceiveConfig
				}
				break
			case StateReceiveConfig:
				err = json.Unmarshal(message, &configMessage)
				if err != nil {
					logError(BTSTRP_LOG, "Unmarshal config message error: %s", err)
				}

				// Send message that configuration was successfully retrieved
				state = StateReceiveReady
				sendConfigConfirmationMsg(conn, id)
				break
			case StateReceiveReady:
				readyMessage := new(ReceiveTypeMessage)
				logDebug(BTSTRP_LOG, string(message))
				err = json.Unmarshal(message, &readyMessage)
				if err != nil {
					logFatal(BTSTRP_LOG, "Unmarshal ready message error: %s", err)
				}

				sendReadyMessage(conn, id)

				// Change state into ready. This terminates connection from websocket server
				// and node starts to function on its own, i.e., change into p2p mode
				logDebug(BTSTRP_LOG, "This node is ready to pass into p2p mode")

				state = StateFinish
				readyToClose = true
				conn.Close()
				break
			case StateFinish:
				return
			default:
				panic("unhandled default case")
			}
			logDebug(BTSTRP_LOG, "Received message: %s", message)
		}
	}()

	// Send hello message
	state = StateReceiveHelloConfirm
	sendHelloMsg(conn, id)

	for {
		select {
		case <-done:
			return &configMessage.Data
		case <-interrupt:
			logInfo(BTSTRP_LOG, "Interrupting connection...")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logError(BTSTRP_LOG, "Error closing connection: %s", err)
				return nil
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func sendHelloMsg(conn *websocket.Conn, selfId string) {
	jsonMsg, _ := json.Marshal(SendIdentityMessage{Type: "hello", Id: selfId})
	err := conn.WriteMessage(websocket.TextMessage, jsonMsg)
	if err != nil {
		logError(BTSTRP_LOG, "Error sending hello message: %s", err)
		return
	}
}

func sendConfigConfirmationMsg(conn *websocket.Conn, selfId string) {
	jsonMsg, _ := json.Marshal(SendIdentityMessage{Type: "conf", Id: selfId})
	err := conn.WriteMessage(websocket.TextMessage, jsonMsg)
	if err != nil {
		logError(BTSTRP_LOG, "Error sending config confirmation message: %s", err)
		return
	}
}

func sendReadyMessage(conn *websocket.Conn, selfId string) {
	jsonMsg, _ := json.Marshal(SendIdentityMessage{Type: "ready", Id: selfId})
	err := conn.WriteMessage(websocket.TextMessage, jsonMsg)
	if err != nil {
		logError(BTSTRP_LOG, "Error sending ready message: %s", err)
		return
	}
}
