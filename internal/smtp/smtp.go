package smtp

import (
	"log/slog"
	"net"

	"github.com/Queueue0/jums/internal/smtp/packets"
)

func Handle(c net.Conn) {
	defer c.Close()
	slog.Debug("handling connection...", "addr", c.RemoteAddr().String())
	Send(packets.NewStatus(220, "Josh's Unremarkable Mail Server v0.0.0"), c)
	s := NewSession(c)
	for s.Open() {
		s.HandleNextLine()
	}
}

func Send(p packets.Packet, c net.Conn) error {
	slog.Debug("Sending packet", "to", c.RemoteAddr().String(), "msg", p.SafeString())
	_, err := c.Write(p.Bytes())
	return err
}
