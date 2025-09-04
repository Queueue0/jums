package smtp

import (
	"log/slog"
	"net"

	"github.com/Queueue0/jums/internal/smtp/mail"
)

type Session struct {
	state  state
	open   bool
	conn   net.Conn
	name   string
	ext    bool
	authed bool
	mail   *mail.Mail
}

func NewSession(c net.Conn) *Session {
	s := &Session{
		open:   true,
		conn:   c,
		name:   "",
		ext:    false,
		authed: false,
		mail:   nil,
	}

	s.state = newState(s)
	return s
}

func (s *Session) Open() bool {
	return s.open
}

func (s *Session) HandleNextLine() error {
	b, err := s.readLine()
	if err != nil {
		return err
	}

	resp := s.state.Handle(b)
	if resp == nil {
		return nil
	}

	if resp.Code() % 100 == 21 {
		s.open = false
	}

	return Send(resp, s.conn)
}

func (s *Session) SendMail() error {
	return nil
}

func (s *Session) readLine() ([]byte, error) {
	read := []byte{}
	for len(read) < 2 || string(read[len(read)-2:]) != "\r\n" {
		next := make([]byte, 1)
		_, err := s.conn.Read(next)
		if err != nil {
			return nil, err
		}
		read = append(read, next...)
	}

	slog.Debug("line received", "addr", s.conn.RemoteAddr().String(), "line", string(read))
	return read, nil
}

