package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kpiotrowski/distributed-monitor"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

const maxBufSize = 3

type prodCons struct {
	buffer    *[]int
	monitor   *monitor.DMonitor
	condFull  *monitor.DCond
	condEmpty *monitor.DCond
	name      string
}

func (pc *prodCons) produce() {
	pc.monitor.Lock()
	defer pc.monitor.UnLock()

	// log.Info("BufProducent: ", pc.buffer)
	for len(*pc.buffer) == maxBufSize {
		fmt.Printf("%s waiting on FULL\n", pc.name)
		pc.condFull.Wait()
	}

	d := rand.Int()
	*pc.buffer = append(*pc.buffer, d)
	fmt.Printf("%s produced %d\n", pc.name, d)

	pc.condEmpty.Signal()
}

func (pc *prodCons) consume() {
	pc.monitor.Lock()
	defer pc.monitor.UnLock()

	// log.Info("BufPKonsument: ", pc.buffer)
	for len(*pc.buffer) == 0 {
		fmt.Printf("%s waiting on EMPTY\n", pc.name)
		pc.condEmpty.Wait()
	}

	x := (*pc.buffer)[0]
	*pc.buffer = (*pc.buffer)[1:]
	fmt.Printf("%s consumed %d\n", pc.name, x)

	pc.condFull.Signal()
}

func main() {
	log.SetLevel(log.DebugLevel)
	data := []int{}
	if len(os.Args) < 2 {
		panic("Not enough arguments to run. You should execute example with [this_node_addr] [node1_addr] [node2_addr] ...")
	}

	cluster, err := monitor.NewCluster(os.Args[1], os.Args[2:]...)
	if err != nil {
		log.Error("Failed to create cluster: ", err)
		return
	}

	monitor, err := cluster.NewMonitor("monitorProdCons")

	if err != nil {
		log.Error("Failed tocreate distributed monitor: ", err)
		return
	}
	monitor.BindData("buforek", &data)
	prodCons := prodCons{
		buffer:    &data,
		condEmpty: monitor.NewCond("condEmpty"),
		condFull:  monitor.NewCond("condFull"),
		monitor:   monitor,
		name:      os.Args[1],
	}
	cluster.Start()

	if os.Args[1] == "127.0.0.1:5556" {
		// go func() {
		for {
			prodCons.produce()
			// time.Sleep(time.Millisecond * 500)
		}
		// }()
	} else {
		for {
			prodCons.consume()
			time.Sleep(time.Millisecond * 500)
		}
	}
}
