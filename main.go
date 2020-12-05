package main

import (
	"./helper"
	"./types"
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

var running bool
var logger *log.Logger

func main() {
	logger = log.New(os.Stdout, "", 0)

	clientChannels := initCluster()
	running = true
	for running {
		logger.Println("Welcome to message sender. Please input all the messages and type SEND when all messages are been inputted.")

		messages := make([]types.ClientMessageRequest, 0)

		for true {
			commandName := helper.ReadString("Start New Message or send messages (start/send):")
			if commandName == "send" {
				break
			}
			sendOutOfOrder := helper.ReadBool("Should the messages be sent out of order?")
			senderId := helper.ReadInt("Sender Id:")
			messageBodies := make([]string, 0)
			for true {
				messageBody := helper.ReadString(fmt.Sprintf("Message body (or \"done\"): "))
				if messageBody == "done" {
					break
				} else {
					messageBodies = append(messageBodies, messageBody)
				}
			}
			newMessage := types.ClientMessageRequest{senderId, messageBodies, sendOutOfOrder}
			messages = append(messages, newMessage)
		}

		for _, request := range messages {
			clientChannels[request.SenderId] <- request
		}
		time.Sleep(10 * time.Second)
	}
}

func shuffleMessages(messages []types.Message) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(messages), func(i, j int) { messages[i], messages[j] = messages[j], messages[i] })
}

func broadcastMessage(senderId int, channels *ClusterChannels, message types.Message) {
	for channelIndex, channel := range channels {
		if channelIndex != senderId {
			channel <- message
		}
	}
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
	serverLog := make([]types.Message, 0)
	globalLog := make([]types.Message, 0)

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
					Timestamp: serverState.LocalTime,
				}
				messages = append(messages, message)
				*serverState.LocalLog = append(*serverState.LocalLog, message)
				*serverState.GlobalLog = addToGlobalLog(serverState.GlobalLog, message)
				logger.Println(serverState.Id, "global message log", *serverState.GlobalLog)
				logger.Println(ServerFlag, serverState.Id, "broadcasting", message)
			}

			if request.IsShuffled {
				shuffleMessages(messages)
				logger.Println("Final send order:", messages)
			}

			for _, message := range messages {
				broadcastMessage(serverState.Id, clusterChannels, message)
				//time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// message handler for given server
func clusterMessagesHandler(serverState *types.ServerState, clusterChannels *ClusterChannels) {
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
				resendIds := getResendIds(message.Sender, serverState.Id, serverState.LocalTime, message.Timestamp)
				lt := getTime2(serverState.LocalTime, serverState.Id)
				mt := getTime2(message.Timestamp, serverState.Id)

				if len(resendIds) == 0 { // no resends needed
					if mt > lt { //ignore stale messages
						serverState.LocalTime[message.Sender] = message.Timestamp[message.Sender]
						logger.Println(serverState.Id, "'s local time: ", serverState.LocalTime)
						*serverState.GlobalLog = addToGlobalLog(serverState.GlobalLog, message)
						logger.Println(serverState.Id, "global message log", *serverState.GlobalLog)
					}
				} else  {
					resendMessage := types.Message{
						Type:      types.ResendM,
						Sender:    serverState.Id,
						Timestamp: serverState.LocalTime,
					}
					for _, id := range resendIds {
						clusterChannels[id] <- resendMessage
						logger.Println(ServerFlag, serverState.Id, "requested", resendMessage, " from ", id)
					}
					*messageQueue = append(*messageQueue, message) //messageQueue to process when our clock is caught up.
				}
			} else if message.Type == types.ResendM {
				requestedTime := message.Timestamp[serverState.Id] + 1

				for _, m := range *serverState.LocalLog {
					if m.Timestamp[serverState.Id] == requestedTime {
						clusterChannels[message.Sender] <- m
						logger.Println(ServerFlag, serverState.Id, "resent", m)
						break
					}
				}
			}
			*messageQueue = (*messageQueue)[1:]
		}
		//time.Sleep(time.Millisecond * 100)
	}
}

func getResendIds(sender int, receiver int, localTime [CLUSTER_SIZE]int, timestamp [CLUSTER_SIZE]int) []int {
	resendIds := make([]int, 0)

	for id, t := range localTime {
		if id != receiver && t < timestamp[id] {
			if id == sender && t < timestamp[id]-1 {
				resendIds = append(resendIds, id)
			} else if id != sender && t < timestamp[id] {
				resendIds = append(resendIds, id)
			}
		}
	}

	return resendIds
}


func addToGlobalLog(globalLog *[]types.Message, message types.Message) []types.Message {
	log := *globalLog

	if len(log) == 0 {
		log = append(log, message)
		return log
	} else {
		for j:=0; j<len(log); j++ {
			entry := log[j]
			if vecEquals(entry.Timestamp, message.Timestamp) {
				return log
			}
		}
		for i := len(log) - 1; i >= 0; i-- {
			entry := log[i]
			if getTime(entry.Timestamp) <= getTime(message.Timestamp) {
				if len(log) - 1 == i {
					log = append(log, message)
					return log
				} else {
					log = append(log[:i+1], log[i:]...)
					log[i+1] = message
					return log
				}
			}
		}

		return log
	}
}

func getTime(t [CLUSTER_SIZE]int) int {
	time := 0

	for i := 0; i < CLUSTER_SIZE; i++ {
		time += t[i]
	}

	return time
}

func getTime2(t [CLUSTER_SIZE]int, exclude int) int {
	time := 0

	for i := 0; i < CLUSTER_SIZE; i++ {
		if i!=exclude {
			time += t[i]
		}
	}

	return time
}

func vecEquals(one [CLUSTER_SIZE]int, two [CLUSTER_SIZE]int) bool{
	for i:=0; i<CLUSTER_SIZE; i++ {
		if one[i] != two[i] {
			return false
		}
	}
	return true
}