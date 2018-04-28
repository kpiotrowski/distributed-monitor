package monitor

import (
	"encoding/json"
	"fmt"

	zmq "github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
)

const (
	msgTypeREQUEST = "MSG_REQUEST"
	msgTypeTOKEN   = "MSG_TOKEN"
	msgTypePingREQ = "MSG_PING_REQ"
	msgTypePingRES = "MSG_PING_RES"
	msgTypeWait    = "MSG_WAIT"
	msgTypeSignal  = "MSG_SIGNAL"
)

type messageDetails struct {
	MsgType string `json:"type"`
	Payload []byte `json:"payload"`
	Monitor string `json:"monitor"`
}

type pingMessage struct {
	Sender string `json:"sender"`
}

type requestMessage struct {
	Sender         string `json:"sender"`
	SequenceNumber int    `json:"seq_num"`
}

type condMessage struct {
	Sender string `json:"sender"`
	Cond   string `json:"cond"`
}

func receiveMessages(cluster *Cluster) {
	receiver, err := zmq.NewSocket(zmq.PULL)
	defer receiver.Close()
	if err != nil {
		log.Error("Failed to create socket: ", err)
	}

	receiver.Bind(fmt.Sprintf("tcp://*:%s", cluster.port))
	for cluster.working {
		data, err := receiver.RecvBytes(0)

		if err != nil {
			log.Error("Failed receive data: ", err)
			continue
		}
		err = handleMessageData(cluster, data)
		if err != nil {
			log.Error("Failed to handle message: ", err)
		}
	}
}

func handleMessageData(cluster *Cluster, message []byte) error {
	msgDet := messageDetails{}
	err := json.Unmarshal(message, &msgDet)
	if err != nil {
		return err
	}

	switch msgDet.MsgType {
	case msgTypeREQUEST:
		return handleRequestMessage(cluster.monitors[msgDet.Monitor], msgDet.Payload)
	case msgTypeTOKEN:
		return handleTokenMessage(cluster.monitors[msgDet.Monitor], msgDet.Payload)
	case msgTypePingREQ:
		return handlePingReqMessage(cluster, msgDet.Payload)
	case msgTypePingRES:
		return handlePingRespMessage(cluster, msgDet.Payload)
	case msgTypeWait:
		return handleWaitMessage(cluster.monitors[msgDet.Monitor], msgDet.Payload)
	case msgTypeSignal:
		return handleSigalMessage(cluster.monitors[msgDet.Monitor], msgDet.Payload)
	}
	return nil
}

func handleWaitMessage(monitor *DMonitor, message []byte) error {
	body := condMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	monitor.reqCSMutex.Lock()
	log.Debugf("Received Wait message from %s, cond: %s", body.Sender, body.Cond)
	monitor.Conds[body.Cond].waitingNodes = append(monitor.Conds[body.Cond].waitingNodes, body.Sender)
	monitor.reqCSMutex.Unlock()

	return nil
}

func handleSigalMessage(monitor *DMonitor, message []byte) error {
	body := condMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	monitor.reqCSMutex.Lock()
	log.Debugf("Received signal messageon %s", body.Cond)

	if elemInArray(monitor.cluster.HostAddr, monitor.Conds[body.Cond].waitingNodes) {
		monitor.Conds[body.Cond].waitChannel <- true
		monitor.Conds[body.Cond].waitingNodes = removeElemArray(monitor.cluster.HostAddr, monitor.Conds[body.Cond].waitingNodes)
	}

	monitor.reqCSMutex.Unlock()

	return nil
}

func handleRequestMessage(monitor *DMonitor, message []byte) error {
	body := requestMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}

	monitor.reqCSMutex.Lock()
	defer monitor.reqCSMutex.Unlock()

	if monitor.requestNumber[body.Sender] > body.SequenceNumber {
		return nil
	}
	// log.Info("received REQUEST message: ", body)

	monitor.requestNumber[body.Sender] = body.SequenceNumber

	if !monitor.cSbusy && monitor.token != nil && monitor.token.LastRequestNumber[body.Sender]+1 == monitor.requestNumber[body.Sender] {
		monitor.token.saveData(monitor.Data)
		monitor.token.Sender = monitor.cluster.HostAddr
		log.Debug("Sending token to: ", body.Sender)
		err = sendTokenMessage(monitor.Name, monitor.cluster.HostAddr, monitor.cluster.Nodes[body.Sender], monitor.token)
		monitor.token = nil
	}

	return err
}

func handleTokenMessage(monitor *DMonitor, message []byte) error {
	body := tokenStr{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	log.Debug("received TOKEN from: ", body.Sender)
	monitor.tokenChannel <- body
	return nil
}

func handlePingReqMessage(cluster *Cluster, message []byte) error {
	// log.Info("Ping req")
	body := pingMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	return sendPingResponse(cluster.HostAddr, cluster.Nodes[body.Sender])
}

func handlePingRespMessage(cluster *Cluster, message []byte) error {
	// log.Info("Ping resp")
	body := pingMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	cluster.pingChannel <- body.Sender
	return nil
}

func sendPingMessage(sender string, socket *zmq.Socket) error {
	payload, _ := json.Marshal(pingMessage{Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypePingREQ,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}

func sendPingResponse(sender string, socket *zmq.Socket) error {
	payload, _ := json.Marshal(pingMessage{Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypePingRES,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}

func sendCsRequestMessage(monitor, sender string, socket *zmq.Socket, seqNumber int) error {
	x := requestMessage{Sender: sender, SequenceNumber: seqNumber}
	payload, _ := json.Marshal(&x)

	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypeREQUEST,
		Payload: payload,
		Monitor: monitor,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}

func sendTokenMessage(monitor, sender string, socket *zmq.Socket, token *tokenStr) error {
	payload, _ := json.Marshal(token)
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypeTOKEN,
		Payload: payload,
		Monitor: monitor,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}

func sendWaitingMessage(monitor, cond, sender string, socket *zmq.Socket) error {
	payload, _ := json.Marshal(condMessage{Cond: cond, Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypeWait,
		Monitor: monitor,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}

func sendSignalMessage(monitor, cond, sender string, socket *zmq.Socket) error {
	payload, _ := json.Marshal(condMessage{Cond: cond, Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypeSignal,
		Monitor: monitor,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq.DONTWAIT)
	return err
}
