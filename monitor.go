package monitor

import (
	"sync"

	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	listenPort = 5678
)

type DMonitor struct {
	Name          string
	mutex         *sync.Mutex
	reqCSMutex    *sync.Mutex
	Conds         map[string]*DCond
	Data          map[string]interface{}
	requestNumber map[string]int
	cSbusy        bool
	cSwaiting     bool
	tokenChannel  chan tokenStr

	token   *tokenStr
	cluster *Cluster
}

//CreateDMonitor created new distributed monitor, test connection and start receiving messages
func createMonitor(cluster *Cluster, name, host string, hosts []string) (*DMonitor, error) {
	dMonitor := DMonitor{
		Name:          name,
		mutex:         &sync.Mutex{},
		reqCSMutex:    &sync.Mutex{},
		Data:          map[string]interface{}{},
		Conds:         map[string]*DCond{},
		requestNumber: map[string]int{host: 0},
		cSbusy:        false,
		cSwaiting:     false,
		cluster:       cluster,
		tokenChannel:  make(chan tokenStr),
	}

	for _, h := range hosts {
		dMonitor.requestNumber[h] = 0
	}

	hosts = append(hosts, host)
	var selecttedHost string
	for _, h := range hosts {
		if selecttedHost == "" || h < selecttedHost {
			selecttedHost = h
		}
	}
	if selecttedHost == host {
		dMonitor.token = createFirstToken(hosts)
	}

	return &dMonitor, nil
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

	dMonitor.token.saveData(dMonitor.Data)
	dMonitor.releaseCS()

	dMonitor.reqCSMutex.Unlock()
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
func (dMonitor *DMonitor) BindData(name string, data interface{}) {
	dMonitor.Data[name] = data
}

func (dMonitor *DMonitor) requestCS() {
	dMonitor.requestNumber[dMonitor.cluster.HostAddr]++
	for k, n := range dMonitor.cluster.Nodes {
		log.Debug("Sending CS request: ", k)
		err := sendCsRequestMessage(dMonitor.Name, dMonitor.cluster.HostAddr, n, dMonitor.requestNumber[dMonitor.cluster.HostAddr])
		if err != nil {
			log.Error(err)
		}
	}
	token := <-dMonitor.tokenChannel
	dMonitor.token = &token //waiting on tooken
}

func (dMonitor *DMonitor) releaseCS() {
	dMonitor.token.LastRequestNumber[dMonitor.cluster.HostAddr] = dMonitor.requestNumber[dMonitor.cluster.HostAddr]
	for node := range dMonitor.cluster.Nodes {
		if !elemInArray(node, dMonitor.token.WaitinQ) && dMonitor.requestNumber[node] == dMonitor.token.LastRequestNumber[node]+1 {
			dMonitor.token.WaitinQ = append(dMonitor.token.WaitinQ, node)
		}
	}
	if len(dMonitor.token.WaitinQ) > 0 {
		nodeToSend := dMonitor.token.WaitinQ[0]
		dMonitor.token.WaitinQ = dMonitor.token.WaitinQ[1:]
		log.Debug("Sending token to: ", nodeToSend)
		sendTokenMessage(dMonitor.Name, dMonitor.cluster.HostAddr, dMonitor.cluster.Nodes[nodeToSend], dMonitor.token)
		dMonitor.token = nil
	}
}

func elemInArray(el string, arr []string) bool {
	for _, e := range arr {
		if e == el {
			return true
		}
	}
	return false
}
