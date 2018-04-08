package monitor

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	listenPort = 5678
)

type DMonitor struct {
	port          string
	mutex         *sync.Mutex
	reqCSMutex    *sync.Mutex
	Conds         map[string]*DCond
	Data          map[string]interface{}
	HostAddr      string
	Nodes         map[string]*zmq.Socket
	requestNumber map[string]int
	cSbusy        bool
	cSwaiting     bool
	working       bool
	pingChannel   chan string
	tokenChannel  chan *tokenStr

	token *tokenStr
}

//CreateDMonitor created new distributed monitor, test connection and start receiving messages
func CreateDMonitor(host string, hosts ...string) (*DMonitor, error) {
	dMonitor := DMonitor{
		port:          strings.Split(host, ":")[1],
		mutex:         &sync.Mutex{},
		reqCSMutex:    &sync.Mutex{},
		HostAddr:      host,
		Data:          map[string]interface{}{},
		Conds:         map[string]*DCond{},
		requestNumber: map[string]int{host: 0},
		cSbusy:        false,
		cSwaiting:     false,
		working:       true,
		pingChannel:   make(chan string),
		Nodes:         map[string]*zmq.Socket{},
	}

	for _, h := range hosts {
		dMonitor.Nodes[h], _ = zmq.NewSocket(zmq.PUSH)
		dMonitor.Nodes[h].Connect(fmt.Sprintf("tcp://%s", h))
		dMonitor.requestNumber[h] = 0
	}

	go dMonitor.receiveMessages()
	dMonitor.testConnection()

	return &dMonitor, nil
}

//Destroy function stops monitor
func (dMonitor *DMonitor) Destroy() {
	dMonitor.working = false
	//TODO IMPLEMENT DESTROY
}

//Lock implements a distributed lock operation
func (dMonitor *DMonitor) Lock() {
	dMonitor.mutex.Lock()
	dMonitor.reqCSMutex.Lock()

	dMonitor.cSwaiting = true

	if dMonitor.token != nil {
		dMonitor.cSbusy = true
		dMonitor.cSwaiting = false
		dMonitor.reqCSMutex.Unlock()
		return
	}

	dMonitor.reqCSMutex.Unlock()

	dMonitor.requestCS()

	dMonitor.reqCSMutex.Lock()
	dMonitor.cSbusy = true
	dMonitor.cSwaiting = false
	dMonitor.token.loadData(&dMonitor.Data)

	dMonitor.reqCSMutex.Unlock()
}

//UnLock implements a distributed unlock operation
func (dMonitor *DMonitor) UnLock() {
	dMonitor.reqCSMutex.Lock()
	dMonitor.cSbusy = false
	dMonitor.cSwaiting = false

	dMonitor.reqCSMutex.Unlock()

	dMonitor.token.saveData(dMonitor.Data)
	//TODO IMPLEMENT DISTRIBUTED UNLOCK

	dMonitor.mutex.Unlock()
}

//NewCond created and returns new conditional variable
func (dMonitor *DMonitor) NewCond() *DCond {
	condNameU, _ := uuid.NewV4()
	cond := createDCond(dMonitor, condNameU.String())
	dMonitor.Conds[condNameU.String()] = cond

	return cond
}

//BindData binds data variable with monitor structure. data should be a pointer to the variable
func (dMonitor *DMonitor) BindData(data interface{}) {
	dataNameU, _ := uuid.NewV4()
	dMonitor.Data[dataNameU.String()] = data
}

func (dMonitor *DMonitor) requestCS() {
	dMonitor.requestNumber[dMonitor.HostAddr]++

	//TODO WAIT ON TOKEN, TOKEN SHOULD BE ADDED TO STRUCT

	*dMonitor.token = *<-dMonitor.tokenChannel //waiting on tooken
}

func (dMonitor *DMonitor) releaseCS() {

}

func (dMonitor *DMonitor) receiveMessages() {

	receiver, err := zmq.NewSocket(zmq.PULL)
	defer receiver.Close()
	if err != nil {
		log.Error("Failed to create socket: ", err)
	}

	receiver.Bind(fmt.Sprintf("tcp://*:%s", dMonitor.port))
	for dMonitor.working {
		data, err := receiver.RecvBytes(0)

		if err != nil {
			log.Error("Failed receive data: ", err)
			continue
		}
		err = handleMessageData(dMonitor, data)
		if err != nil {
			log.Error("Failed to handle message: ", err)
		}
	}
}

func (dMonitor *DMonitor) testConnection() error {
	for k, n := range dMonitor.Nodes {
		ch := make(chan bool)

		log.Info("Sending ping: ", k)

		err := sendPingMessage(dMonitor.HostAddr, n)
		if err != nil {
			log.Error(err)
		}
		var ping string

		go func() {

			select {
			case ping = <-dMonitor.pingChannel:
				ch <- true
			case <-time.After(5 * time.Second):
				ch <- false
			}

		}()
		if x := <-ch; x == false {
			log.Error("Ping timed out!")
			return errors.New("Ping timed out")
		}
		log.Info("Received ping: ", ping)
	}
	return nil
}
