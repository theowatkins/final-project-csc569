package types

type MessageType string
const RegularM MessageType = "REGULAR"
const ResendM MessageType = "RESEND"

const CLUSTER_SIZE = 3

type Message struct {

	// Regular or resend
	Type MessageType

	// REGULAR: Text being sent
	// RESEND: Empty if the message is requesting a resend
	Body string

	// REGULAR: Whoever is sending the message
	// RESEND: Whoever is sending the resend request
	Sender int

	// REGULAR: Timestamp of the text associated with the message
	// RESEND: Timestamp of the message being requested
	Timestamp [CLUSTER_SIZE]int
}


type ServerState struct {
	Id        int
	LocalLog  * []Message
	GlobalLog * []string
	LocalTime [CLUSTER_SIZE]int

}

type ClientMessageRequest struct {
	MessageBodies []string
	OutOfOrder bool
}