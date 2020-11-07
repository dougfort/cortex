package main

import (
	"log"
	"math/rand"
	"sync"
)

type NodeID uint32

type Event struct {
	SourceID NodeID
	Text     string
}

type Message struct {
	DestID  NodeID
	Path    []NodeID
	Content string
}

type ReceiverNode struct {
	ID      NodeID
	MsgChan chan<- Message
}

type Node struct {
	ID           NodeID
	EventChan    chan<- Event
	IncomingChan <-chan Message
	DeliveryChan chan<- Message
	Receivers    []ReceiverNode
}

func main() {
	const NODE_COUNT = 100
	const MESSAGE_COUNT = 10

	log.Printf("program starts with %d nodes to deliver %d messages", NODE_COUNT, MESSAGE_COUNT)

	deliveryChan := make(chan Message, 1)

	eventChan := make(chan Event, NODE_COUNT)
	eventsDone := make(chan struct{})
	go func() {
		reportEvents(eventChan, eventsDone)
	}()

	nodeIDs := make([]NodeID, NODE_COUNT)
	msgChans := make([]chan Message, NODE_COUNT)
	receivers := make([]ReceiverNode, NODE_COUNT)
	for i := 0; i < NODE_COUNT; i++ {
		nodeIDs[i] = computeID(i)
		msgChans[i] = make(chan Message, 1)
		receivers[i] = ReceiverNode{
			ID:      nodeIDs[i],
			MsgChan: msgChans[i],
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < NODE_COUNT; i++ {
		node := Node{
			ID:           nodeIDs[i],
			EventChan:    eventChan,
			IncomingChan: msgChans[i],
			DeliveryChan: deliveryChan,
			Receivers:    receivers,
		}
		wg.Add(1)
		go func() {
			nodeFunc(node)
			wg.Done()
		}()
	}

	// send some messages, asynchronously
	go func() {

		for i := 0; i < MESSAGE_COUNT; i++ {
			// pick a random destination Node
			destID := nodeIDs[rand.Int()%len(nodeIDs)]

			// send a message through a randomly selected msgChan
			msgChan := msgChans[rand.Int()%len(msgChans)]
			msgChan <- Message{
				DestID: destID,
			}
		}
	}()

	// wait for all messages to be received
	var receivedCount int
RECEIVE_LOOP:
	for msg := range deliveryChan {
		log.Printf("%05d: message received %s %v", msg.DestID, msg.Content, msg.Path)
		receivedCount++
		if receivedCount == MESSAGE_COUNT {
			log.Println("all messages received")
			break RECEIVE_LOOP
		}
	}

	// close all the msg chans
	for _, msgChan := range msgChans {
		close(msgChan)
	}

	log.Println("waiting for nodes to finish")
	wg.Wait()

	log.Println("draining events")
	close(eventChan)
	<-eventsDone

	log.Println("program ends")
}

func computeID(i int) NodeID {
	return NodeID(uint32(i + 1))
}

func nodeFunc(node Node) {
	node.EventChan <- Event{SourceID: node.ID, Text: "start"}
	for msg := range node.IncomingChan {
		// if this message is to us, it is delivered
		if msg.DestID == node.ID {
			node.DeliveryChan <- msg
		} else {
			// pass the message on to a random receiver
			receiver := node.Receivers[rand.Int()%len(node.Receivers)]
			msg.Path = append(msg.Path, node.ID)
			receiver.MsgChan <- msg
		}
	}
	node.EventChan <- Event{SourceID: node.ID, Text: "end"}
}

func reportEvents(eventChan <-chan Event, eventsDone chan<- struct{}) {
	for event := range eventChan {
		log.Printf("%05d: %s", event.SourceID, event.Text)
	}
	log.Println("eventChan closed")
	close(eventsDone)
}
