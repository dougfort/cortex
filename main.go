package main

import (
	"flag"
	"log"
	"math/rand"
	"sync"
)

type Config struct {
	NodeCount    int
	MessageCount int
}

type NodeID uint32

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
	IncomingChan <-chan Message
	DeliveryChan chan<- Message
	Receivers    []ReceiverNode
}

func main() {
	config := loadConfig()

	log.Printf("program starts with %d nodes to deliver %d messages", config.NodeCount, config.MessageCount)

	deliveryChan := make(chan Message, 1)

	nodeIDs := make([]NodeID, config.NodeCount)
	msgChans := make([]chan Message, config.NodeCount)
	receivers := make([]ReceiverNode, config.NodeCount)
	for i := 0; i < config.NodeCount; i++ {
		nodeIDs[i] = computeID(i)
		msgChans[i] = make(chan Message, 1)
		receivers[i] = ReceiverNode{
			ID:      nodeIDs[i],
			MsgChan: msgChans[i],
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < config.NodeCount; i++ {
		node := Node{
			ID:           nodeIDs[i],
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

		for i := 0; i < config.MessageCount; i++ {
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
		if receivedCount == config.MessageCount {
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

	log.Println("program ends")
}

func loadConfig() Config {
	var config Config
	flag.IntVar(&config.NodeCount, "node-count", 10, "Number of nodes")
	flag.IntVar(&config.MessageCount, "message-count", 1, "Number of messages sent between nodes")
	flag.Parse()
	return config
}

func computeID(i int) NodeID {
	return NodeID(uint32(i + 1))
}

func nodeFunc(node Node) {
	for msg := range node.IncomingChan {
		// if this message is to us, it is delivered
		if msg.DestID == node.ID {
			node.DeliveryChan <- msg
		} else {
			// pass the message on to a random receiver, but not ourselves
		RECEIVER_LOOP:
			for {
				receiver := node.Receivers[rand.Int()%len(node.Receivers)]
				if receiver.ID == node.ID {
					continue RECEIVER_LOOP
				}

				msg.Path = append(msg.Path, node.ID)
				receiver.MsgChan <- msg
				break RECEIVER_LOOP
			}
		}
	}
}
