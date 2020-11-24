package main

import (
	"./types"
	"log"
)
import "fmt"

const CLUSTER_SIZE = types.CLUSTER_SIZE

// early todos:
//  - How are we going to do client communication? (who's sending messages)
//  - What do we do with a message when it's processed?

func main() {
    init_cluster()
    fmt.Println("done")
}

func init_cluster() {
	var com_chans [CLUSTER_SIZE]chan types.Message

	// Create communication channels for cluster
	for i:=0; i<CLUSTER_SIZE; i++ {
		com_chans[i] = make(chan types.Message)
		go start_server(i, &com_chans)
	}
}

func start_server(id int, com_chans *[CLUSTER_SIZE]chan types.Message) {
    var time [CLUSTER_SIZE]int
    log := make([]string, 0)

    // Make state
    server_state := types.ServerState{
        id, // Server ID
        log, // Empty message log
        time, // Vector time initialized to 0
    }

	// TODO: send messages (broadcast) and update vector time
	go receive_messages(&server_state, com_chans)
}

// message handler for given server
func receive_messages(server_state *types.ServerState, com_chans *[CLUSTER_SIZE]chan types.Message) {
	select {
	case message := <-com_chans[server_state.Id]:
		if message.Type == types.RegularM {
			ourTime := server_state.LocalTime[message.Sender]
			messageTime := message.Timestamp
			expectedMessageTime := ourTime + 1

			if messageTime > expectedMessageTime {
				resendMessage := types.Message{
					types.ResendM,
					"",
					server_state.Id,
					expectedMessageTime,
				}
				com_chans[server_state.Id] <- resendMessage
			} else if messageTime == expectedMessageTime { // message as expected, update local time and save to log
				server_state.LocalTime[message.Sender] = message.Timestamp
				server_state.MessageLog = append(server_state.MessageLog, message.Message)
				fmt.Println("Server ", server_state.Id, "received message: \n", message.Message, "\n")
			} //ignore broadcasts messages for resends

		} else if message.Type == types.ResendM {
			if message.Timestamp < 0 || message.Timestamp >= len(server_state.MessageLog) {
				log.Fatal("Invalid timestamp requested:", message.Timestamp)
				return
			}

			requestedMessage := types.Message{
				types.RegularM,
				server_state.MessageLog[message.Timestamp],
				server_state.Id,
				message.Timestamp,
			}
			com_chans[message.Sender] <- requestedMessage
		}
	}
}