package main

import "./types"
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

	// Set up communication channels
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
        time} // Vector time initialized to 0

	// TODO: send messages (broadcast) and update vector time
	go receive_messages(&server_state, com_chans)
}

func receive_messages(server_state *types.ServerState, com_chans *[CLUSTER_SIZE]chan types.Message) {
	select {
	case message := <-com_chans[server_state.Id]:
		if message.Type == types.RegularM {
            // TODO: out of order messages

            // Update local time
            server_state.LocalTime[message.Sender] = message.Timestamp[message.Sender]

            // Print message for now
            // TODO: what are we gonna do with messages?
            fmt.Println("Server ", server_state.Id, "received message: ")
            fmt.Println(message.Message, "\n")
		} else if message.Type == types.ResendM {
            // TODO: look into log and find message with correct timestamp (if it exists)
		}   
	}
}