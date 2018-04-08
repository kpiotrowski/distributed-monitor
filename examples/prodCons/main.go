package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kpiotrowski/distributed-monitor"
	"golang.org/x/exp/rand"
)

const maxBufSize = 3

type prodCons struct {
	buffer    *[]int
	monitor   *monitor.DMonitor
	condFull  *monitor.DCond
	condEmpty *monitor.DCond
}

func (pc *prodCons) produce() {
	pc.monitor.Lock()
	defer pc.monitor.UnLock()

	for len(*pc.buffer) == maxBufSize {
		fmt.Printf("%s waiting on FULL\n", "NONAME")
		pc.condFull.Wait()
	}

	d := rand.Int()
	*pc.buffer = append(*pc.buffer, d)
	fmt.Printf("%s produced %d\n", "NONAME", d)

	pc.condEmpty.Signal()
}

func (pc *prodCons) consume() {
	pc.monitor.Lock()
	defer pc.monitor.UnLock()

	for len(*pc.buffer) == 0 {
		fmt.Printf("%s waiting on EMPTY\n", "NONAME")
		pc.condEmpty.Wait()
	}

	x := (*pc.buffer)[0]
	*pc.buffer = (*pc.buffer)[1:]
	fmt.Printf("%s condumed %d\n", "NONAME", x)

	pc.condFull.Signal()
}

func main() {
	data := []int{}
	if len(os.Args) < 2 {
		panic("Not enough arguments to run. You should execute example with [this_node_addr] [node1_addr] [node2_addr] ...")
	}
	monitor, err := monitor.CreateDMonitor(os.Args[1], os.Args[2:]...)
	if err != nil {
		print("Failed tocreate distributed monitor: ", err)
		return
	}
	monitor.BindData(&data)
	prodCons := prodCons{
		buffer:    &data,
		condEmpty: monitor.NewCond(),
		condFull:  monitor.NewCond(),
		monitor:   monitor,
	}
	go func() {
		for {
			prodCons.produce()
			time.Sleep(time.Millisecond * 300)
		}
	}()
	for {
		prodCons.consume()
		time.Sleep(time.Millisecond * 500)
	}
}
