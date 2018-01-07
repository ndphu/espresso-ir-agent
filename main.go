package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ndphu/espresso-commons"
	"github.com/ndphu/espresso-commons/messaging"
	"github.com/ndphu/espresso-commons/model/event"
	"github.com/ndphu/espresso-ir-agent/lirc"
	"log"
	"strconv"
)

var (
	DefaultLircdHost                           = "192.168.1.22:8765"
	DefaultBrokerHost                          = "19november.freeddns.org"
	DefaultBrokerPort                          = "5384"
	ReconnectTimeout                           = 5
	IREventChannel    chan (event.IREvent)     = nil
	MessageRouter     *messaging.MessageRouter = nil
	DeviceSerial                               = ""
	EventTopic        messaging.Topic          = ""
)

func main() {
	// mqtt
	mqttHost := commons.GetEnv("MSG_BROKER_HOST", DefaultBrokerHost)
	mqttPort, _ := strconv.Atoi(commons.GetEnv("MSG_BROKER_PORT", DefaultBrokerPort))

	MessageRouter, err := messaging.NewMessageRouter(mqttHost, mqttPort, "", "", fmt.Sprintf("ir-agent-%d", commons.GetRandom()))
	if err != nil {
		panic(err)
	}
	defer MessageRouter.Stop()

	if MessageRouter.GetMQTTClient().IsConnected() {
		log.Println("MQTT connected!")
	} else {
		panic(errors.New("MQTT not connected"))
	}

	// Device serial
	if commons.GetEnv("PLATFORM", "") == "rpi" {
		DeviceSerial, err = commons.RPiGetSerial("/proc/cpuinfo")
		if err != nil {
			panic(err)
		}
	} else {
		DeviceSerial = "development"
	}
	log.Printf("Device Serial: [%s]\n", DeviceSerial)

	EventTopic = messaging.Topic(fmt.Sprintf("eps/device/%s/event/ir", DeviceSerial))

	// lirc
	IREventChannel = make(chan (event.IREvent), 1024)

	lircdHost := commons.GetEnv("LIRCD_HOST", DefaultLircdHost)
	lircApp, err := lirc.NewLirc(lircdHost, IREventChannel, ReconnectTimeout)
	if err != nil {
		panic(err)
	}
	lircApp.Start()

	for {
		msg := event.IREvent(<-IREventChannel)
		// ignore repeated event
		if msg.Repeat == 0 {
			log.Println("Got IR message")
			b, _ := json.Marshal(msg)
			log.Printf("%s\n", string(b))
			eventMsg := messaging.Message{
				Destination: EventTopic,
				Type:        "IR_EVENT",
				Source:      "ir-agent",
				Payload:     string(b),
			}
			err := MessageRouter.Publish(eventMsg)
			if err != nil {
				log.Println("Messenger failed to publish message: ", err)
			} else {
				log.Println("Published.")
			}
		}
	}
	log.Println("End")
}
