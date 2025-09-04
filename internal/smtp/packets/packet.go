package packets

import "net"

type Packet interface {
	Bytes() []byte
	String() string
	SafeString() string
	Send(net.Conn) error
}
