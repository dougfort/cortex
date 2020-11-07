package main

import (
	"log"
	"sync"
)

type Event struct {
	NodeID uint32
	Text   string
}

type Message struct {
	SenderID uint32
	Content  string
}

type ReceiverNode struct {
	ID      uint32
	MsgChan chan<- Message
}

type Node struct {
	ID        uint32
	EventChan chan<- Event
	MsgChan   <-chan Message
	receivers []ReceiverNode
}

func main() {
	log.Println("program starts")

	const count = 2

	eventChan := make(chan Event, count)
	eventsDone := make(chan struct{})
	go func() {
		reportEvents(eventChan, eventsDone)
	}()

	msgChans := make([]chan Message, count)
	receivers := make([]ReceiverNode, count)
	for i := 0; i < count; i++ {
		msgChans[i] = make(chan Message, 1)
		receivers[i] = ReceiverNode{
			ID:      computeID(i),
			MsgChan: msgChans[i],
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		node := Node{
			ID:        computeID(i),
			EventChan: eventChan,
			MsgChan:   msgChans[i],
			receivers: receivers,
		}
		wg.Add(1)
		go func() {
			nodeFunc(node)
			wg.Done()
		}()
	}

	log.Println("waiting for nodes to finish")
	wg.Wait()

	log.Println("draining events")
	close(eventChan)
	<-eventsDone

	log.Println("program ends")
}

func computeID(i int) uint32 {
	return uint32(i + 1)
}

func nodeFunc(node Node) {
	node.EventChan <- Event{NodeID: node.ID, Text: "start"}
	node.EventChan <- Event{NodeID: node.ID, Text: "end"}
}

func reportEvents(eventChan <-chan Event, eventsDone chan<- struct{}) {
	for event := range eventChan {
		log.Printf("%05d: %s", event.NodeID, event.Text)
	}
	log.Println("eventChan closed")
	close(eventsDone)
}
