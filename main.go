package main

import (
	"../final/helper"
	"../final/types"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

const ServerFlag = "**"
const CLUSTER_SIZE = types.CLUSTER_SIZE

type MessageChannel = chan types.Message
type ClusterChannels = [CLUSTER_SIZE]MessageChannel
type ClientChannel = chan types.ClientMessageRequest
type ClientChannels = [CLUSTER_SIZE]ClientChannel

// early todos:
//  - How are we going to do client communication? (who's sending messages)
//  - What do we do with a message when it's processed?
var running bool
var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "", 0)

	clientChannels := initCluster()
	running = true
	for running {
		logger.Println("Welcome to message sender.")
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

func broadcastMessage(senderId int, channels *ClusterChannels, message types.Message) {
	for channelIndex, channel := range channels {
		if channelIndex != senderId {
			channel <- message
		}
	}
}

func shuffleMessage(messages []types.Message) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(messages), func(i, j int) { messages[i], messages[j] = messages[j], messages[i] })
}

func initCluster() ClientChannels {
	var clusterChannels ClusterChannels
	var clientChannels ClientChannels

	// Create communication channels for cluster
	for i := 0; i < CLUSTER_SIZE; i++ {
		clusterChannels[i] = make(MessageChannel)
		clientChannels[i] = make(ClientChannel)
		go startServer(i, &clusterChannels, &clientChannels[i])
	}
	return clientChannels
}

func startServer(id int, clusterChannels *ClusterChannels, clientChannels *ClientChannel) {
	var serverTime [CLUSTER_SIZE]int
	serverLog := make([]string, 0)
	globalLog := make([]string, 0)

	for i := 0; i < len(serverTime); i++ {
		serverTime[i] = -1 // cause we expect next message to be this + 1
	}

	// Make state
	serverState := types.ServerState{
		Id:        id,         // Server ID
		LocalLog:  &serverLog, // Empty message log
		GlobalLog: &globalLog,
		LocalTime: serverTime, // Vector time initialized to 0,
	}

	go clientRequestHandler(&serverState, clusterChannels, clientChannels)
	go clusterMessagesHandler(&serverState, clusterChannels)
}

func clientRequestHandler(serverState *types.ServerState, clusterChannels *ClusterChannels, clientChannel *ClientChannel) {
	for {
		select {
		case request := <-*clientChannel:
			messages := make([]types.Message, 0)
			for _, messageBody := range request.MessageBodies {
				serverState.LocalTime[serverState.Id]++
				message := types.Message{
					Type:      types.RegularM,
					Body:      messageBody,
					Sender:    serverState.Id,
					Timestamp: serverState.LocalTime[serverState.Id],
				}
				messages = append(messages, message)
				*serverState.LocalLog = append(*serverState.LocalLog, message.Body)
			}

			if request.OutOfOrder {
				shuffleMessage(messages)
			}
			for _, message := range messages {
				logger.Println(ServerFlag, serverState.Id, "broadcasting", message)
				broadcastMessage(serverState.Id, clusterChannels, message)
			}
		}
	}
}

// message handler for given server
func clusterMessagesHandler(serverState *types.ServerState, clusterChannels *ClusterChannels, ) {
	messageQueue := make([]types.Message, 0)

	go func() { //load messages into the queue
		for {
			select {
			case message := <-clusterChannels[serverState.Id]:
				logger.Println(ServerFlag, serverState.Id, "received", message)
				messageQueue = append(messageQueue, message)
			}
		}
	}()

	go processMessageQueue(serverState, clusterChannels, &messageQueue)
}

func processMessageQueue(serverState *types.ServerState, clusterChannels *ClusterChannels, messageQueue *[]types.Message) {

	for {
		if len(*messageQueue) > 0 {
			message := (*messageQueue)[0]
			if message.Type == types.RegularM {
				expectedMessageTime := serverState.LocalTime[message.Sender] + 1

				if message.Timestamp == expectedMessageTime { // message as expected, update local time and save to log
					serverState.LocalTime[message.Sender] = message.Timestamp
					*serverState.GlobalLog = append(*serverState.GlobalLog, message.Body)
					logger.Println(serverState.Id, "global message log", *serverState.GlobalLog)
				} else if message.Timestamp > expectedMessageTime {
					resendMessage := types.Message{
						Type:      types.ResendM,
						Sender:    serverState.Id,
						Timestamp: expectedMessageTime,
					}
					clusterChannels[message.Sender] <- resendMessage
					*messageQueue = append(*messageQueue, message) //messageQueue to process when our clock is caught up.

					logger.Println(ServerFlag, serverState.Id, "requested", resendMessage)
				}
			} else if message.Type == types.ResendM {
				if message.Timestamp < 0 || message.Timestamp >= len(*serverState.LocalLog) {
					log.Fatal("Invalid timestamp requested:", message.Timestamp)
				}
				messageTime := message.Timestamp
				requestedMessage := types.Message{
					Type:      types.RegularM,
					Body:      (*serverState.LocalLog)[messageTime],
					Sender:    serverState.Id,
					Timestamp: messageTime,
				}
				clusterChannels[message.Sender] <- requestedMessage
				logger.Println(ServerFlag, serverState.Id, "resent", requestedMessage)
			}
			*messageQueue = (*messageQueue)[1:]
		}
		time.Sleep(time.Millisecond * 100)
	}
}
