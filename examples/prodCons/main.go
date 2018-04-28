package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kpiotrowski/distributed-monitor"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

var logLevel = log.InfoLevel

const maxBufSize = 3

type prodCons struct {
	buffer    *[]int
	monitor   *monitor.DMonitor
	condFull  *monitor.DCond
	condEmpty *monitor.DCond
	name      string
}

func (pc *prodCons) produce() {
	pc.monitor.Lock()         //Lock
	defer pc.monitor.UnLock() //Unlock when func ends

	for len(*pc.buffer) == maxBufSize {
		fmt.Printf("%s waiting on FULL\n", pc.name)
		pc.condFull.Wait() //Wait on condFull variable
	}

	d := rand.Int()
	*pc.buffer = append(*pc.buffer, d)
	fmt.Printf("%s produced %d\n", pc.name, d)

	pc.condEmpty.Signal() //Signal on condEmpty variable
}

func (pc *prodCons) consume() {
	pc.monitor.Lock()         //Lock
	defer pc.monitor.UnLock() //Unlock whe nfunc ends

	for len(*pc.buffer) == 0 {
		fmt.Printf("%s waiting on EMPTY\n", pc.name)
		pc.condEmpty.Wait() //Wait on condEmpty variable
	}

	x := (*pc.buffer)[0]
	*pc.buffer = (*pc.buffer)[1:]
	fmt.Printf("%s consumed %d\n", pc.name, x)

	pc.condFull.Signal() //Signal on condFull example
}

func main() {
	log.SetLevel(logLevel)
	data := []int{}
	if len(os.Args) < 2 {
		panic("Not enough arguments to run. You should execute example with [this_node_addr] [node1_addr] [node2_addr] ...")
	}

	cluster, err := monitor.NewCluster(os.Args[1], os.Args[2:]...) //Create new claster - communication service
	if err != nil {
		log.Error("Failed to create cluster: ", err)
		return
	}
	defer cluster.Destroy()

	monitor, err := cluster.NewMonitor("monitorProdCons") //Add new monitor to your cluster

	if err != nil {
		log.Error("Failed tocreate distributed monitor: ", err)
		return
	}
	monitor.BindData("buforek", &data) //Bind bata with your monitor

	prodCons := prodCons{
		buffer:    &data,
		condEmpty: monitor.NewCond("condEmpty"), //Create new conditional variable
		condFull:  monitor.NewCond("condFull"),  //Create new conditional variable
		monitor:   monitor,
		name:      os.Args[1],
	}
	cluster.Start()

	if os.Args[1] == "127.0.0.1:5551" {
		//go func() {
		for {
			prodCons.produce()
			time.Sleep(time.Millisecond * 200)
		}
		//}()
	} else {
		for {
			prodCons.consume()
			time.Sleep(time.Millisecond * 500)
		}
	}
}
