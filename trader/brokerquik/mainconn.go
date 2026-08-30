package brokerquik

import (
	"bufio"
	"log"
	"net"

	"ergo.services/ergo/gen"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type mainConnection struct {
	gen.MetaProcess
	// Чтобы логировать запросы/ответы к внешним API без экранирования символов.
	logger  *log.Logger
	tcpConn net.Conn
	writer  *transform.Writer
}

func createMainConnection(port int) (gen.MetaBehavior, error) {
	tcpConn, err := dial(port)
	if err != nil {
		return nil, err
	}
	var quikCharmap = charmap.Windows1251
	return &mainConnection{
		tcpConn: tcpConn,
		writer:  transform.NewWriter(tcpConn, quikCharmap.NewEncoder()),
	}, nil
}

func (conn *mainConnection) Init(process gen.MetaProcess) error {
	conn.MetaProcess = process
	return nil
}

func (conn *mainConnection) Start() error {
	defer conn.Log().Debug("finish")
	var quikCharmap = charmap.Windows1251
	var reader = bufio.NewReader(transform.NewReader(conn.tcpConn, quikCharmap.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if len(incoming) < 2_048 {
			conn.Log().Trace("response %v", incoming)
		}
		message := messageTCP{
			Data: []byte(incoming),
		}
		if err := conn.Send(conn.Parent(), message); err != nil {
			return err
		}
	}
}

func (conn *mainConnection) HandleMessage(from gen.PID, message any) error {
	messageTcp, ok := message.(messageTCP)
	if !ok {
		return nil
	}
	// при сетевой ошибке supervisor сможет перезапустить брокера
	if _, err := conn.writer.Write(messageTcp.Data); err != nil {
		return err
	}
	//TODO if !bytes.HasSuffix()
	if _, err := conn.writer.Write([]byte("\r\n")); err != nil {
		return err
	}
	//conn.Log().Trace("request %v", string(messageTcp.Data))
	return nil
}

func (conn *mainConnection) HandleCall(from gen.PID, ref gen.Ref, request any) (any, error) {
	return gen.ErrUnsupported, nil
}

func (conn *mainConnection) Terminate(reason error) {
	conn.tcpConn.Close()
}

func (conn *mainConnection) HandleInspect(from gen.PID, item ...string) map[string]string {
	return map[string]string{}
}
