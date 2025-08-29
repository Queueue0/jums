package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/mail"
	"github.com/Queueue0/jums/internal/packets"
)

type smtpState interface {
	Conn() net.Conn
	Handle([]byte) (smtpState, *packets.Status)
}

func newState(c net.Conn) smtpState {
	return &baseState{c}
}

// Generic functions for EHLO and HELO because all states basically treat them
// the same
func ehlo(s smtpState, c *packets.Command) (smtpState, *packets.Status) {
	if len(c.Args()) < 1 {
		return s, packets.NewStatus(501, "Syntax error, tell me who you are!")
	}

	name := c.Args()[0]

	var gs *greetedState
	if _, ok := s.(*greetedState); ok {
		gs = s.(*greetedState)
	} else {
		gs = &greetedState{s.Conn(), name, true}
	}

	var sts *packets.Status
	if _, ok := s.Conn().(*tls.Conn); ok {
		sts = packets.NewStatus(250, fmt.Sprintf("Hello there, %s!", name))
	} else {
		sts = packets.NewStatus(250, fmt.Sprintf("Hello there, %s!", name), "STARTTLS")
	}

	return gs, sts
}

func helo(s smtpState, c *packets.Command) (smtpState, *packets.Status) {
	if len(c.Args()) < 1 {
		return s, packets.NewStatus(501, "Syntax error, tell me who you are!")
	}
	name := c.Args()[0]

	var gs *greetedState
	if _, ok := s.(*greetedState); ok {
		gs = s.(*greetedState)
	} else {
		gs = &greetedState{s.Conn(), name, false}
	}

	return gs, packets.NewStatus(250, fmt.Sprintf("Hello there, %s", name))
}

type baseState struct {
	c net.Conn
}

func (s *baseState) Conn() net.Conn {
	return s.c
}

func (s *baseState) Handle(b []byte) (smtpState, *packets.Status) {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		return ehlo(s, c)
	case "HELO":
		//TODO: verify domain to prevent spam
		return helo(s, c)
	case "MAIL":
		// No mail until we've been greeted
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "RCPT":
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "DATA":
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "RSET":
		return s, packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return s, packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return s, packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return s, verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := s.c.(*tls.Conn); ok {
			return s, packets.NewStatus(454, "TLS already in use")
		}

		tlsState, err := startTLS(s.c)
		if err != nil {
			return s, packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}

		fmt.Println("TLS handshake successful")
		return tlsState, nil
	default:
		return s, packets.NewStatus(500, "command unrecoginized")
	}
}

func startTLS(c net.Conn) (*baseState, error) {
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
	return &baseState{tlsc}, nil
}

type greetedState struct {
	c    net.Conn
	name string
	ext  bool
}

func (s *greetedState) Conn() net.Conn {
	return s.c
}

func (s *greetedState) Handle(b []byte) (smtpState, *packets.Status) {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		return ehlo(s, c)
	case "HELO":
		//TODO: verify domain to prevent spam
		return helo(s, c)
	case "MAIL":
		if !strings.Contains(c.Args()[0], ":") {
			return s, packets.NewStatus(501, "Syntax error")
		}
		fparts := strings.Split(c.Args()[0], ":")
		if len(fparts) != 2 {
			return s, packets.NewStatus(501, "Syntax error")
		}
		fparts[0] = strings.ToUpper(fparts[0])
		if fparts[0] != "FROM" {
			return s, packets.NewStatus(501, "Syntax error")
		}
		if !strings.HasPrefix(fparts[1], "<") || !strings.HasSuffix(fparts[1], ">") {
			return s, packets.NewStatus(501, "Syntax error")
		}
		fstr := strings.Trim(fparts[1], "<>")
		from, err := mail.NewAddress(fstr)
		if err != nil {
			return s, packets.NewStatus(553, "Invalid sender mailbox name (format should be user@domain)")
		}

		m := &mail.Mail{
			From: from,
			Rcpt: []mail.Address{},
			Data: []byte{},
		}

		return &rcptState{s.c, s.name, s.ext, m}, packets.NewStatus(250, "OK proceed")
	case "RCPT":
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "DATA":
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "RSET":
		return s, packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return s, packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return s, packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return s, verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := s.c.(*tls.Conn); ok {
			return s, packets.NewStatus(454, "TLS already in use")
		}

		tlsState, err := startTLS(s.c)
		if err != nil {
			return s, packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}

		return tlsState, nil

	default:
		return s, packets.NewStatus(500, "command unrecoginized")
	}
}

type rcptState struct {
	c    net.Conn
	name string
	ext  bool
	m    *mail.Mail
}

func (s *rcptState) Conn() net.Conn {
	return s.c
}

func (s *rcptState) Handle(b []byte) (smtpState, *packets.Status) {
	c := packets.ParseCommand(b)
	switch c.Cmd() {
	case "EHLO":
		// TODO: verify the given domain/address literal to prevent spam
		return ehlo(s, c)
	case "HELO":
		//TODO: verify domain to prevent spam
		return helo(s, c)
	case "MAIL":
		return s, packets.NewStatus(503, "Bad sequence of commands")
	case "RCPT":
		rs, err := parseTO(c.Args()[0])
		if err != nil {
			return s, packets.NewStatus(501, "Syntax error")
		}

		ra, err := mail.NewAddress(rs)
		if err != nil {
			return s, packets.NewStatus(550, fmt.Sprintf("Invalid address %s", rs))
		}

		s.m.Rcpt = append(s.m.Rcpt, *ra)
		return s, packets.NewStatus(250, fmt.Sprintf("RCPT <%s> OK", rs))
	case "DATA":
		return &dataState{s.c, s.name, s.ext, s.m}, packets.NewStatus(354, "Start mail input; end with <CRLF>.<CRLF>")
	case "RSET":
		return &greetedState{s.c, s.name, s.ext}, packets.NewStatus(250, "Reset OK")
	case "NOOP":
		return s, packets.NewStatus(250, "NOOP OK")
	case "QUIT":
		return s, packets.NewStatus(221, "Goodbye!")
	case "VRFY":
		return s, verify(c.Args()[0])
	case "STARTTLS":
		if _, ok := s.c.(*tls.Conn); ok {
			return s, packets.NewStatus(454, "TLS already in use")
		}

		tlsState, err := startTLS(s.c)
		if err != nil {
			return s, packets.NewStatus(421, "TLS handshake failed, terminating connection")
		}

		return tlsState, nil

	default:
		return s, packets.NewStatus(500, "command unrecoginized")
	}
}

type dataState struct {
	c    net.Conn
	name string
	ext  bool
	m    *mail.Mail
}

func (s *dataState) Conn() net.Conn {
	return s.c
}

func (s *dataState) Handle(b []byte) (smtpState, *packets.Status) {
	s.m.Data = append(s.m.Data, b...)
	if bytes.Equal(b, []byte(".\r\n")) {
		s.generateReceived()
		go func() {
			s.m.Send()
		}()
		return &greetedState{s.c, s.name, s.ext}, packets.NewStatus(250, "OK")
	}
	return s, nil
}

func (s *dataState) generateReceived() {
	var rname string
	remote, _, err := net.SplitHostPort(s.c.RemoteAddr().String())
	if err != nil {
		remote = s.c.RemoteAddr().String()
	}
	rnames, err := net.LookupAddr(remote)
	if err != nil {
		rname = ""
	} else {
		rname = rnames[0]
	}

	from := fmt.Sprintf("from %s (%s [%s])", s.name, rname, remote)

	var smtpType string
	if s.ext {
		smtpType = "ESMTP"
		if isTls(s.c) {
			smtpType += "S"
		}
	} else {
		smtpType = "SMTP"
	}

	tlsInfo := ""
	if isTls(s.c) {
		tlsState := s.c.(*tls.Conn).ConnectionState()
		vn := tls.VersionName(tlsState.Version)
		csn := tls.CipherSuiteName(tlsState.CipherSuite)
		tlsInfo = fmt.Sprintf("(version=%s cipher=%s)", vn, csn)
	}

	conf := config.GetConfig()
	by := fmt.Sprintf("by %s", conf.Mxdomain)
	with := fmt.Sprintf("with %s", smtpType)
	timestamp := time.Now().Local().Format("Mon, 02 Jan 2006 15:04:05 -0700 (MST)")

	s.m.GenerateId()
	id := fmt.Sprintf("id %s", s.m.Id)
	s.m.Received = mail.PartialReceived{
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
