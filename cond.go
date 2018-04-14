package monitor

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
	dCond.monitor.UnLock()

	//TODO notify other nodes that this node is waiting
	<-dCond.waitChannel

	dCond.monitor.Lock()
}

//Signal wakes up one of the waiting nodes
//If there is no waiting nodes signal is not sent
func (dCond *DCond) Signal() {
	//TODO implement distriuted wakeup signal
	go dCond.wakeUp()
}

func (dCond *DCond) wakeUp() {
	if len(dCond.waitingNodes) > 0 {
		dCond.waitingNodes = dCond.waitingNodes[1:]
		dCond.waitChannel <- true //TODO remove this
	}
}
