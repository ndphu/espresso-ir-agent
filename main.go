package main

import (
	"fmt"
	"github.com/ndphu/espresso-commons"
	"github.com/ndphu/espresso-commons/dao"
	"github.com/ndphu/espresso-commons/messaging"
	"github.com/ndphu/espresso-commons/model"
	"github.com/ndphu/espresso-commons/repo"
	"github.com/ndphu/espresso-ir-agent/lirc"
	"gopkg.in/mgo.v2"
	"os"
)

var (
	DefaultLircdHost                          = "127.0.0.1:8765"
	ReconnectTimeout                          = 5
	IREventChannel   chan (model.IRMessage)   = nil
	MessageRouter    *messaging.MessageRouter = nil
)

func main() {
	// database
	fmt.Println("Connecting to db...")
	s, err := mgo.Dial("127.0.0.1:27017")

	if err != nil {
		fmt.Println("Fail to connect to DB")
		panic(err)
	} else {
		fmt.Println("Connected to DB")
	}

	irRepo := &repo.IREventRepo{
		Session: s,
	}

	// end database

	// mqtt
	MessageRouter, err = messaging.NewMessageRouter("127.0.0.1", 1883, "", "", fmt.Sprintf("ir-agent-%d", commons.GetRandom()))
	if err != nil {
		panic(err)
	}
	defer MessageRouter.Stop()

	// lirc
	IREventChannel = make(chan (model.IRMessage), 1024)

	lircdHost := os.Getenv("LIRCD_HOST")
	if len(lircdHost) == 0 {
		lircdHost = DefaultLircdHost
	}

	lircApp, err := lirc.NewLirc(lircdHost, IREventChannel, ReconnectTimeout)
	if err != nil {
		panic(err)
	}
	lircApp.Start()

	for {
		msg := model.IRMessage(<-IREventChannel)
		fmt.Println("Got IR message. Inserting...")
		err := dao.Insert(irRepo, &msg)
		if err != nil {
			fmt.Println("Fail to insert event to db. This event will be discarded")
			fmt.Println(err)
		} else {
			PushEvent(msg)
		}
	}

	fmt.Println("End")

}

func PushEvent(msg model.IRMessage) {
	eventMsg := model.Message{
		Destination: commons.IRAgentEventTopic,
		Type:        "IR_EVENT_ADDED",
		Source:      "ir-agent",
		Payload:     msg.Id,
	}
	err := MessageRouter.Publish(eventMsg)
	if err != nil {
		fmt.Println("Messenger failed to publish message", err)
	} else {
		fmt.Println("Published IR_EVENT_ADDED")
	}
}
