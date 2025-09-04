package smtp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/smtp/mail"
	"github.com/Queueue0/jums/internal/smtp/packets"
)

type state interface {
	session() *Session
	Handle([]byte) *packets.Status
}

func newState(s *Session) state {
	return &baseState{s}
}

// Generic functions for EHLO and HELO because all states basically treat them
// the same
func ehlo(s state, c *packets.Command) (state, *packets.Status) {
	if len(c.Args()) < 1 {
		return s, packets.NewStatus(501, "Syntax error, tell me who you are!")
	}

	name := c.Args()[0]

	var gs *greetedState
	if _, ok := s.(*greetedState); ok {
		gs = s.(*greetedState)
	} else {
		gs = &greetedState{s.session()}
	}

	var sts *packets.Status
	if _, ok := s.session().conn.(*tls.Conn); ok {
		sts = packets.NewStatus(250, fmt.Sprintf("Hello there, %s!", name), "AUTH PLAIN")
	} else {
		sts = packets.NewStatus(250, fmt.Sprintf("Hello there, %s!", name), "STARTTLS")
	}

	s.session().ext = true

	return gs, sts
}

func helo(s state, c *packets.Command) (state, *packets.Status) {
	if len(c.Args()) < 1 {
		return s, packets.NewStatus(501, "Syntax error, tell me who you are!")
	}
	name := c.Args()[0]

	var gs *greetedState
	if _, ok := s.(*greetedState); ok {
		gs = s.(*greetedState)
	} else {
		gs = &greetedState{s.session()}
	}

	s.session().ext = false

	return gs, packets.NewStatus(250, fmt.Sprintf("Hello there, %s", name))
}

type baseState struct {
	s *Session
}

func (st *baseState) session() *Session {
	return st.s
}

func (st *baseState) Handle(b []byte) *packets.Status {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		ns, resp := ehlo(st, c)
		st.s.state = ns
		return resp
	case "HELO":
		//TODO: verify domain to prevent spam
		ns, resp := helo(st, c)
		st.s.state = ns
		return resp
	case "MAIL":
		// No mail until we've been greeted
		return packets.NewStatus(503, "Bad sequence of commands")
	case "RCPT":
		return packets.NewStatus(503, "Bad sequence of commands")
	case "DATA":
		return packets.NewStatus(503, "Bad sequence of commands")
	case "RSET":
		return packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := st.s.conn.(*tls.Conn); ok {
			return packets.NewStatus(454, "TLS already in use")
		}

		tlsc, err := startTLS(st.s.conn)
		if err != nil {
			return packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}
		st.s.conn = tlsc

		return nil
	default:
		return packets.NewStatus(500, "command unrecoginized")
	}
}

func startTLS(c net.Conn) (*tls.Conn, error) {
	_, err := c.Write(packets.NewStatus(220, "OK").Bytes())
	tlsc := tls.Server(c, &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			conf := config.GetConfig()
			cert, err := tls.LoadX509KeyPair(conf.CertFile, conf.KeyFile)
			if err != nil {
				return nil, err
			}
			return &cert, nil
		},
	})
	err = tlsc.Handshake()
	if err != nil {
		return nil, err
	}
	return tlsc, nil
}

type greetedState struct {
	s *Session
}

func (st *greetedState) session() *Session {
	return st.s
}

func (st *greetedState) Handle(b []byte) *packets.Status {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		ns, resp := ehlo(st, c)
		st.s.state = ns
		return resp
	case "HELO":
		//TODO: verify domain to prevent spam
		ns, resp := helo(st, c)
		st.s.state = ns
		return resp
	case "MAIL":
		if !strings.Contains(c.Args()[0], ":") {
			return packets.NewStatus(501, "Syntax error")
		}
		fparts := strings.Split(c.Args()[0], ":")
		if len(fparts) != 2 {
			return packets.NewStatus(501, "Syntax error")
		}
		fparts[0] = strings.ToUpper(fparts[0])
		if fparts[0] != "FROM" {
			return packets.NewStatus(501, "Syntax error")
		}
		if !strings.HasPrefix(fparts[1], "<") || !strings.HasSuffix(fparts[1], ">") {
			return packets.NewStatus(501, "Syntax error")
		}
		fstr := strings.Trim(fparts[1], "<>")
		from, err := mail.NewAddress(fstr)
		if err != nil {
			return packets.NewStatus(553, "Invalid sender mailbox name (format should be user@domain)")
		}

		st.s.mail = &mail.Mail{
			From: from,
			Rcpt: []mail.Address{},
			Data: []byte{},
		}

		st.s.state = &rcptState{st.s}
		return packets.NewStatus(250, "OK proceed")
	case "RCPT":
		return packets.NewStatus(503, "Bad sequence of commands")
	case "DATA":
		return packets.NewStatus(503, "Bad sequence of commands")
	case "RSET":
		return packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := st.s.conn.(*tls.Conn); ok {
			return packets.NewStatus(454, "TLS already in use")
		}

		tlsc, err := startTLS(st.s.conn)
		if err != nil {
			return packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}
		st.s.conn = tlsc

		return nil

	default:
		return packets.NewStatus(500, "command unrecoginized")
	}
}

type rcptState struct {
	s *Session
}

func (st *rcptState) session() *Session {
	return st.s
}

func (st *rcptState) Handle(b []byte) *packets.Status {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		ns, resp := ehlo(st, c)
		st.s.state = ns
		return resp
	case "HELO":
		//TODO: verify domain to prevent spam
		ns, resp := helo(st, c)
		st.s.state = ns
		return resp
	case "MAIL":
		return packets.NewStatus(503, "Bad sequence of commands")
	case "RCPT":
		rs, err := parseTO(c.Args()[0])
		if err != nil {
			return packets.NewStatus(501, "Syntax error")
		}

		ra, err := mail.NewAddress(rs)
		if err != nil {
			return packets.NewStatus(550, fmt.Sprintf("Invalid address %s", rs))
		}

		config := config.GetConfig()
		if !st.s.authed && ra.Domain != config.Domain {
			return packets.NewStatus(530, "Authentication required for relay")
		}

		st.s.mail.Rcpt = append(st.s.mail.Rcpt, *ra)
		return packets.NewStatus(250, fmt.Sprintf("RCPT <%s> OK", rs))
	case "DATA":
		st.s.state = &dataState{st.s}
		return packets.NewStatus(354, "Start mail input; end with <CRLF>.<CRLF>")
	case "RSET":
		st.s.state = &greetedState{st.s}
		return packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := st.s.conn.(*tls.Conn); ok {
			return packets.NewStatus(454, "TLS already in use")
		}

		tlsc, err := startTLS(st.s.conn)
		if err != nil {
			return packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}
		st.s.conn = tlsc

		return nil

	default:
		return packets.NewStatus(500, "command unrecoginized")
	}
}

type dataState struct {
	s *Session
}

func (st *dataState) session() *Session {
	return st.s
}

func (st *dataState) Handle(b []byte) *packets.Status {
	st.s.mail.Data = append(st.s.mail.Data, b...)
	if bytes.Equal(b, []byte(".\r\n")) {
		st.generateReceived()
		st.s.SendMail()
		st.s.state = &greetedState{st.s}
		return packets.NewStatus(250, "OK")
	}
	return nil
}

func (st *dataState) generateReceived() {
	var rname string
	remote, _, err := net.SplitHostPort(st.s.conn.RemoteAddr().String())
	if err != nil {
		remote = st.s.conn.RemoteAddr().String()
	}
	rnames, err := net.LookupAddr(remote)
	if err != nil {
		rname = ""
	} else {
		rname = rnames[0]
	}

	from := fmt.Sprintf("from %s (%s [%s])", st.s.name, rname, remote)
	enc := isTls(st.s.conn)

	var smtpType string
	if st.s.ext {
		smtpType = "ESMTP"
		if enc {
			smtpType += "S"
		}
	} else {
		smtpType = "SMTP"
	}

	tlsInfo := ""
	if enc {
		tlsState := st.s.conn.(*tls.Conn).ConnectionState()
		vn := tls.VersionName(tlsState.Version)
		csn := tls.CipherSuiteName(tlsState.CipherSuite)
		tlsInfo = fmt.Sprintf("(version=%s cipher=%s)", vn, csn)
	}

	conf := config.GetConfig()
	by := fmt.Sprintf("by %s", conf.Mxdomain)
	with := fmt.Sprintf("with %s", smtpType)
	timestamp := time.Now().Local().Format("Mon, 02 Jan 2006 15:04:05 -0700 (MST)")

	st.s.mail.GenerateId()
	id := fmt.Sprintf("id %s", st.s.mail.Id)
	st.s.mail.Received = mail.PartialReceived{
		From:      from,
		By:        by,
		With:      with,
		TlsInfo:   tlsInfo,
		Id:        id,
		Timestamp: timestamp,
	}
}

// helper functions used only in this file
func isTls(c net.Conn) bool {
	_, ok := c.(*tls.Conn)
	return ok
}

func verify(address string) *packets.Status {
	//TODO actually implement this (could be useful for receiving mail and for authenticated users)
	return packets.NewStatus(252, "VRFY command currently disabled")
}

func parseTO(to string) (string, error) {
	parts := strings.Split(to, ":")
	if len(parts) != 2 {
		return "", errors.New("Syntax error")
	}
	if strings.ToUpper(parts[0]) != "TO" {
		return "", errors.New("Syntax error")
	}
	if !(strings.HasPrefix(parts[1], "<") && strings.HasSuffix(parts[1], ">")) {
		return "", errors.New("Syntax error")
	}

	return strings.Trim(parts[1], "<>"), nil
}
