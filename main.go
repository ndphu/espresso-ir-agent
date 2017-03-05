package main

import (
	"encoding/json"
	"fmt"
	"github.com/ndphu/espresso-commons"
	"github.com/ndphu/espresso-commons/db"
	"github.com/ndphu/espresso-commons/model"
	"github.com/ndphu/espresso-ir-agent/lirc"
	"github.com/ndphu/manga-crawler/dao"
	"gopkg.in/mgo.v2"
	"time"
)

var (
	LircUrl                                 = "192.168.1.21:8765"
	ReconnectTimeout                        = 5
	IREventChannel   chan (model.IRMessage) = nil
)

func main() {
	fmt.Println(time.Now())

	s, err := mgo.Dial("127.0.0.1:27017")
	irRepo := &db.IREventRepo{
		Session:  s,
		Database: s.DB(commons.DBName),
	}
	if err != nil {
		fmt.Println("Fail to connect to DB")
		panic(err)
	} else {
		fmt.Println("Connected to DB")
	}

	IREventChannel = make(chan (model.IRMessage))

	lircApp, err := lirc.NewLirc(LircUrl, IREventChannel, ReconnectTimeout)
	if err != nil {
		panic(err)
	}
	lircApp.Start()

	for {
		msg := model.IRMessage(<-IREventChannel)

		err := dao.Insert(irRepo, &msg)
		if err != nil {
			fmt.Println("Fail to insert event to db", err)
		}

		raw, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Failed to serialize message from lirc app")
		} else {
			fmt.Println("Got message", string(raw), "at", msg.Timestamp)
		}

	}

	fmt.Println("End")

}
