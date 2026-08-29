package brokerquik

import (
	"bufio"
	"encoding/json"
	"finance/model"
	"net"

	"ergo.services/ergo/gen"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type callbackConnection struct {
	gen.MetaProcess
	tcpConn net.Conn
}

func createCallbackConnection(port int) (gen.MetaBehavior, error) {
	tcpConn, err := dial(port)
	if err != nil {
		return nil, err
	}
	return &callbackConnection{
		tcpConn: tcpConn,
	}, nil
}

func (conn *callbackConnection) Init(process gen.MetaProcess) error {
	conn.MetaProcess = process
	return nil
}

func (conn *callbackConnection) Start() error {
	defer conn.Log().Debug("finish")
	var quikCharmap = charmap.Windows1251
	var reader = bufio.NewReader(transform.NewReader(conn.tcpConn, quikCharmap.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		var cj CallbackJson
		err = json.Unmarshal([]byte(incoming), &cj)
		if err != nil {
			return err
		}
		conn.handleCallback(cj)
	}
}

func (conn *callbackConnection) HandleMessage(from gen.PID, message any) error {
	return nil
}

func (conn *callbackConnection) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	return gen.ErrUnsupported, nil
}

func (conn *callbackConnection) Terminate(reason error) {
	conn.tcpConn.Close()
}

func (conn *callbackConnection) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{}
}

func (conn *callbackConnection) handleCallback(cj CallbackJson) {
	// Для эффективности будем обрабатывать только избранные колбеки
	if cj.Command == "NewCandle" {
		if cj.Data == nil {
			return
		}
		var quikCandle Candle
		var err = json.Unmarshal(cj.Data, &quikCandle)
		if err != nil {
			return
		}
		var modelCandle = convertToCandle(quikCandle)
		// TODO можно фильтровать слишком ранние бары
		conn.Send(model.MultyBroker, model.BrokerCallbackMessage{
			Message: modelCandle,
		})
	}
}
