package main

import (
	"../final/types"
	"log"
	"math/rand"
	"time"
	"../final/helper"
)
import "fmt"

const ServerFlag = "**"
const CLUSTER_SIZE = types.CLUSTER_SIZE

type MessageChannel = chan types.Message
type ClusterChannels = [CLUSTER_SIZE]MessageChannel
type ClientChannel = chan types.ClientMessageRequest
type ClientChannels = [CLUSTER_SIZE] ClientChannel
// early todos:
//  - How are we going to do client communication? (who's sending messages)
//  - What do we do with a message when it's processed?
var running bool


func main() {
    clientChannels := init_cluster()
	running = true
    for running {
    	fmt.Println("Welcome to message sender.")
		senderId := helper.ReadInt("Send Id:")
		receiverId := helper.ReadInt("Receiver Id:")
		numberOfMessage := helper.ReadInt("Number of messages: ")
		messageBodies := make([]string, 0)
		for i := 0; i < numberOfMessage; i++ {
			messageBody := helper.ReadString(fmt.Sprintf("Body %d body: ", i))
			messageBodies = append(messageBodies, messageBody)
		}
		sendOutOfOrder := helper.ReadBool("Send out of order (true | false):")
		request := types.ClientMessageRequest{
			ReceiverId:    receiverId,
			MessageBodies: messageBodies,
			OutOfOrder:    sendOutOfOrder,
		}
		clientChannels[senderId] <- request
		time.Sleep(10 * time.Second)
	}
}

func broadcastMessage(senderId int, channels * ClusterChannels, message types.Message){
	for channelIndex, channel := range channels {
		if channelIndex != senderId {
			channel <- message
		}
	}
}

func shuffle_message(messages []types.Message){
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(messages), func(i, j int) { messages[i], messages[j] = messages[j], messages[i] })
}

func init_cluster() ClientChannels {
	var clusterChannels ClusterChannels
	var clientChannels ClientChannels

	// Create communication channels for cluster
	for i:=0; i<CLUSTER_SIZE; i++ {
		clusterChannels[i] = make(MessageChannel)
		clientChannels[i] = make(ClientChannel)
		go start_server(i, &clusterChannels, &clientChannels[i])
	}
	return clientChannels
}

func start_server(id int, clusterChannels *ClusterChannels, clientChannels *ClientChannel) {
    var serverTime [CLUSTER_SIZE]int
    serverLog := make([]string, 0)

    for i := 0 ;i<len(serverTime); i++ {
    	serverTime[i] = -1 // cause we expect next message to be this + 1
	}

    // Make state
    server_state := types.ServerState{
        id, // Server ID
		serverLog, // Empty message log
		serverTime, // Vector time initialized to 0
    }

	go send_message_handler(&server_state, clusterChannels, clientChannels)
	go receive_messages(&server_state, clusterChannels)
}

func send_message_handler(server_state *types.ServerState, com_chans *ClusterChannels, clientChannel *ClientChannel){
	for {
		select {
		case request := <-*clientChannel:
			messages := make([]types.Message, 0)
			for _, messageBody := range request.MessageBodies {
				message := types.Message {
					types.RegularM,
					messageBody,
					server_state.Id,
					server_state.LocalTime[server_state.Id],
				}
				messages = append(messages, message)
				server_state.LocalTime[server_state.Id]++
				server_state.MessageLog = append(server_state.MessageLog, message.Body)
			}

			if request.OutOfOrder {
				shuffle_message(messages)
			}
			for _, message := range messages {
				broadcastMessage(server_state.Id, com_chans, message)
			}
		}
	}
}

// message handler for given server
func receive_messages(server_state *types.ServerState, com_chans *ClusterChannels,) {
	for {
		select {
		case message := <-com_chans[server_state.Id]:
			fmt.Println(ServerFlag, "Server ", server_state.Id, "received message: ", message)

			if message.Type == types.RegularM {
				expectedMessageTime := server_state.LocalTime[message.Sender] + 1
				messageTime := message.Timestamp
				if messageTime > expectedMessageTime {
					fmt.Println(ServerFlag, "Server ", server_state.Id, " expected: ", expectedMessageTime, "got: ", messageTime)
					resendMessage := types.Message{
						types.ResendM,
						"",
						server_state.Id,
						expectedMessageTime,
					}
					com_chans[message.Sender] <- resendMessage
					for messageTime != expectedMessageTime {
						newMessage := <- com_chans[server_state.Id]
						if message.Sender == newMessage.Sender &&
							expectedMessageTime == newMessage.Timestamp {
							messageTime = newMessage.Timestamp // exit is all caught up
						}
					}
				} else if messageTime == expectedMessageTime { // message as expected, update local time and save to log
					server_state.LocalTime[message.Sender] = message.Timestamp
				} //ignore broadcasts messages for resends

			} else if message.Type == types.ResendM {
				fmt.Println(ServerFlag, "Server ", server_state.Id, "received resend request.")
				if message.Timestamp < 0 || message.Timestamp >= len(server_state.MessageLog) {
					log.Fatal("Invalid timestamp requested:", message.Timestamp)
				}

				requestedMessage := types.Message {
					types.RegularM,
					server_state.MessageLog[message.Timestamp],
					server_state.Id,
					message.Timestamp,
				}
				com_chans[message.Sender] <- requestedMessage
			}
		}
	}
}