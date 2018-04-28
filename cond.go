package monitor

import (
	log "github.com/sirupsen/logrus"
)

//DCond ia a distributed conditional variable
type DCond struct {
	monitor      *DMonitor
	name         string
	waitChannel  chan bool
	waitingNodes []string
}

func createDCond(monitor *DMonitor, name string) *DCond {
	cond := DCond{
		monitor:      monitor,
		name:         name,
		waitingNodes: []string{},
	}
	cond.waitChannel = make(chan bool)

	return &cond
}

//Wait function unlocks monitor to allow other nodes to use CS; other nodes are notified about
func (dCond *DCond) Wait() {
	dCond.waitingNodes = append(dCond.waitingNodes, dCond.monitor.cluster.HostAddr)
	for _, soc := range dCond.monitor.cluster.Nodes {
		sendWaitingMessage(dCond.monitor.Name, dCond.name, dCond.monitor.cluster.HostAddr, soc)
	}

	dCond.monitor.UnLock()

	log.Debugf("Waiting on channel %s ", dCond.name)
	<-dCond.waitChannel
	log.Debugf("Channel wake up %s ", dCond.name)

	dCond.monitor.Lock()
}

//Signal wakes up one of the waiting nodes
//If there is no waiting nodes signal is not sent
func (dCond *DCond) Signal() {
	for _, node := range dCond.waitingNodes {
		if node == dCond.monitor.cluster.HostAddr {
			dCond.waitChannel <- true
		} else {
			sendSignalMessage(dCond.monitor.Name, dCond.name, dCond.monitor.cluster.HostAddr, dCond.monitor.cluster.Nodes[node])
		}
	}
	dCond.waitingNodes = []string{}
}
