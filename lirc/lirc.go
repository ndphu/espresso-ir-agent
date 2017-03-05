package lirc

import (
	"github.com/ndphu/espresso-commons/model"
	"github.com/ndphu/lirc"
	"log"
	"time"
)

type Lirc struct {
	LircHost         string
	IREventChannel   chan (model.IRMessage)
	Running          bool
	IRRouter         *lirc.Router
	ReconnectTimeout int
}

func NewLirc(host string, channel chan (model.IRMessage), timeout int) (*Lirc, error) {
	return &Lirc{
		LircHost:         host,
		IREventChannel:   channel,
		Running:          false,
		ReconnectTimeout: timeout,
	}, nil
}

func (l *Lirc) Start() {
	if !l.Running {
		l.Running = true
		go l.loop()
	}
}

func (l *Lirc) Stop() {
	l.Running = false
	l.IRRouter.Close()
}

func (l *Lirc) loop() error {
	for l.Running {
		ir, err := lirc.InitTCP(l.LircHost)

		defer ir.Close()
		if err != nil {
			log.Println("Fail to connect to lircd at", l.LircHost, "error", err)
		} else {
			l.IRRouter = ir
			log.Println("Connected to lircd at", l.LircHost)
			l.IRRouter.Handle("", "", func(e lirc.Event) {
				m := model.IRMessage{
					RemoteName: e.Remote,
					Button:     e.Button,
					Code:       e.Code,
					Repeat:     e.Repeat,
					Source:     l.LircHost,
					Timestamp:  time.Now(),
				}
				l.IREventChannel <- m
			})
			// Run ir
			ir.Run()
		}

		if !l.Running {
			break
		}
		// Wait for 2 seconds and restart
		log.Println("Wait for", l.ReconnectTimeout, "seconds and try to reconnect")
		time.Sleep(time.Duration(l.ReconnectTimeout) * time.Second)
	}
	return nil
}
