package monitor

import (
	"encoding/json"

	"github.com/pebbe/zmq4"
)

const (
	msgTypeREQUEST = "MSG_REQUEST"
	msgTypeTOKEN   = "MSG_TOKEN"
	msgTypePingREQ = "MSG_PING_REQ"
	msgTypePingRES = "MSG_PING_RES"
)

type messageDetails struct {
	MsgType string `json:"type"`
	Payload []byte `json:"payload"`
}

type pingMessage struct {
	Sender string `json:"sender"`
}

func handleMessageData(monitor *DMonitor, message []byte) error {
	msgDet := messageDetails{}
	err := json.Unmarshal(message, &msgDet)
	if err != nil {
		return err
	}

	switch msgDet.MsgType {
	case msgTypeREQUEST:
		return handleRequestMessage(monitor, msgDet.Payload)
	case msgTypeTOKEN:
		return handleTokenMessage(monitor, msgDet.Payload)
	case msgTypePingREQ:
		return handlePingReqMessage(monitor, msgDet.Payload)
	case msgTypePingRES:
		return handlePingRespMessage(monitor, msgDet.Payload)
	}
	return nil
}

func handleRequestMessage(monitor *DMonitor, message []byte) error {
	//TODO -------------------------------
	return nil
}

func handleTokenMessage(monitor *DMonitor, message []byte) error {
	//TODO -------------------------------
	return nil
}

func handlePingReqMessage(monitor *DMonitor, message []byte) error {
	// log.Info("Ping req")
	body := pingMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	return sendPingResponse(monitor.HostAddr, monitor.Nodes[body.Sender])
}

func handlePingRespMessage(monitor *DMonitor, message []byte) error {
	// log.Info("Ping resp")
	body := pingMessage{}
	err := json.Unmarshal(message, &body)
	if err != nil {
		return err
	}
	monitor.pingChannel <- body.Sender
	return nil
}

func sendPingMessage(sender string, socket *zmq4.Socket) error {
	payload, _ := json.Marshal(pingMessage{Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypePingREQ,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq4.DONTWAIT)
	return err
}

func sendPingResponse(sender string, socket *zmq4.Socket) error {
	payload, _ := json.Marshal(pingMessage{Sender: sender})
	msg, _ := json.Marshal(messageDetails{
		MsgType: msgTypePingRES,
		Payload: payload,
	})
	_, err := socket.SendBytes(msg, zmq4.DONTWAIT)
	return err

}
