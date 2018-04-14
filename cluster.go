package monitor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	zmq "github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

// Cluster is structure representing group of machines
type Cluster struct {
	port         string
	HostAddr     string
	Nodes        map[string]*zmq.Socket
	pingChannel  chan string
	tokenChannel chan tokenStr
	working      bool

	monitors map[string]*DMonitor
}

// NewCluster creates new cluster
func NewCluster(host string, hosts ...string) (*Cluster, error) {
	cluster := Cluster{
		port:         strings.Split(host, ":")[1],
		HostAddr:     host,
		pingChannel:  make(chan string),
		tokenChannel: make(chan tokenStr),
		Nodes:        map[string]*zmq.Socket{},
		working:      true,
		monitors:     map[string]*DMonitor{},
	}

	for _, h := range hosts {
		cluster.Nodes[h], _ = zmq.NewSocket(zmq.PUSH)
		cluster.Nodes[h].Connect(fmt.Sprintf("tcp://%s", h))
	}

	go receiveMessages(&cluster)
	err := cluster.testConnection()
	if err != nil {
		return nil, errors.New("Cluster ping timed out")
	}

	return &cluster, nil
}

// Destroy destorys cluster
func (cluster *Cluster) Destroy() {
	cluster.working = false
	for _, soc := range cluster.Nodes {
		soc.Close()
	}
}

func (cluster *Cluster) testConnection() error {
	for k, n := range cluster.Nodes {
		ch := make(chan bool)

		log.Info("Sending ping: ", k)

		err := sendPingMessage(cluster.HostAddr, n)
		if err != nil {
			log.Error(err)
		}
		var ping string

		go func() {
			select {
			case ping = <-cluster.pingChannel:
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

func (cluster *Cluster) NewMonitor(name string) (*DMonitor, error) {
	nodes := []string{}
	for n := range cluster.Nodes {
		nodes = append(nodes, n)
	}
	monitor, err := createMonitor(cluster, name, cluster.HostAddr, nodes)
	if err != nil {
		return nil, err
	}
	cluster.monitors[name] = monitor

	return monitor, nil
}
