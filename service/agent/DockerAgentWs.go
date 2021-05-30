package agent

import (
	"docker-agent/service/conf"
	"docker-agent/utils"
	"encoding/json"
	"log"
	"strconv"
	"time"
)

type DockerAgentWs struct {
}

var wsConn *utils.WsConn

func StartWs() {
	endpoint := conf.DockerWsServer
	wsConn = utils.NewWsBuilder().
		WsUrl(endpoint).
		AutoReconnect().
		ProtoHandleFunc(wsMsgHandle).
		ReconnectInterval(time.Millisecond * 5).
		Build()
	go exitHandler(wsConn)
}

func exitHandler(c *utils.WsConn) {
	pingTicker := time.NewTicker(10 * time.Minute)
	pongTicker := time.NewTicker(time.Second)
	defer pingTicker.Stop()
	defer pongTicker.Stop()
	defer c.CloseWs()

	for {
		select {
		case <-c.CloseChan():
			return
		case t := <-pingTicker.C:
			c.SendPingMessage([]byte(strconv.Itoa(int(t.UnixNano() / int64(time.Millisecond)))))
		case t := <-pongTicker.C:
			c.SendPongMessage([]byte(strconv.Itoa(int(t.UnixNano() / int64(time.Millisecond)))))
		}
	}
}

func SendWsMsg(ch string, data interface{}) {
	msg := map[string]interface{}{
		"ch": ch,
		"ts": time.Now().UnixNano() / 1e6,
		"d":  data,
	}
	wsConn.SendJsonMessage(msg)
}

func wsMsgHandle(msg []byte) error {
	datamap := make(map[string]interface{})
	err := json.Unmarshal(msg, &datamap)
	if err != nil {
		log.Println("json unmarshal error for ", string(msg))
		return err
	}

	ch, isOk := datamap["ch"].(string)
	if !isOk {
		log.Println("no message ch, msg:" + string(msg))
		return err
	}

	log.Println("recv:" + string(msg))
	data := map[string]interface{}{}
	if datamap["d"] != nil {
		data = datamap["d"].(map[string]interface{})
	}
	err, resp := MsgHandle(ch, data)
	SendWsMsg(ch+".ack", map[string]interface{}{"err": err, "resp": resp})
	return err
}
