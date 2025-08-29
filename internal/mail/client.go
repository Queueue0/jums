package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/packets"
)

const (
	initialTimeout         = 5 * time.Minute
	mailTimeout            = 5 * time.Minute
	rcptTimeout            = 5 * time.Minute
	dataInitTimeout        = 2 * time.Minute
	dataBlockTimeout       = 3 * time.Minute
	dataTerminationTimeout = 10 * time.Minute
)

func createSMTPConn(domain string) (net.Conn, error) {
	mxrs, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found mail servers: %v\n", mxrs)

	var c net.Conn
MXLoop:
	for i, mx := range mxrs {
		c, err = connectStartTLS(mx.Host, ":25")
		if err != nil {
			fmt.Printf("Couldn't establish connection: %s\n", err.Error())
			if i == len(mxrs)-1 {
				return nil, err
			}
		} else {
			break MXLoop
		}
	}
	return c, nil
}

// read a potential multi-line response
func readResponse(c net.Conn) ([]byte, error) {
	read := []byte{}
	lastLine := false
	for !lastLine {
		line := []byte{}
		for len(line) < 2 || string(line[len(line)-2:]) != "\r\n" {
			next := make([]byte, 1)
			_, err := c.Read(next)
			if err != nil {
				return nil, err
			}
			line = append(line, next...)
		}
		if line[3] == byte(' ') {
			lastLine = true
		}
		read = append(read, line...)
	}
	fmt.Println(string(read))
	return read, nil
}

// Attempts to connect and elevate to TLS with starttls
// If the server doesn't support starttls it will return a non-encrypted connection
func connectStartTLS(domain, port string) (net.Conn, error) {
	addr := domain + port
	fmt.Printf("dialing %s...\n", addr)
	c, err := net.DialTimeout("tcp", addr, initialTimeout)
	if err != nil {
		return nil, errors.New("connectStartTLS: " + err.Error())
	}

	fmt.Println("reading status...")
	stat, err := readAndParseStatus(c)
	if err != nil {
		c.Close()
		return nil, errors.New("connectStartTLS: " + err.Error())
	}

	if stat.Code() != 220 {
		packets.NewCommand("QUIT").Send(c)
		c.Close()
		return nil, errors.New("connectStartTLS: non-220 connection response: " + stat.String())
	}

	conf := config.GetConfig()
	if err = packets.NewCommand("EHLO", conf.Mxdomain).Send(c); err != nil {
		packets.NewCommand("QUIT").Send(c)
		c.Close()
		return nil, errors.New("connectStartTLS: " + err.Error())
	}

	stat, err = readAndParseStatus(c)
	if err != nil {
		packets.NewCommand("QUIT").Send(c)
		c.Close()
		return nil, errors.New("connectStartTLS: " + err.Error())
	}

	if stat.Code() != 250 {
		// If EHLO unrecognized
		if stat.Code() == 500 {
			err = packets.NewCommand("HELO", conf.Mxdomain).Send(c)
			if err != nil {
				packets.NewCommand("QUIT").Send(c)
				c.Close()
				return nil, errors.New("connectStartTLS: " + err.Error())
			}

			stat, err = readAndParseStatus(c)
			if err != nil {
				packets.NewCommand("QUIT").Send(c)
				c.Close()
				return nil, errors.New("connectStartTLS: " + err.Error())
			}

			if stat.Code() != 250 {
				packets.NewCommand("QUIT").Send(c)
				c.Close()
				return nil, errors.New("connectStartTLS: couldn't greet server: " + stat.String())
			}

			return c, nil
		}
		return nil, errors.New("connectStartTLS: couldn't greet server: " + stat.String())
	}

	if strings.Contains(strings.ToUpper(stat.String()), "STARTTLS") {
		err = packets.NewCommand("STARTTLS").Send(c)
		if err != nil {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: " + err.Error())
		}

		stat, err := readAndParseStatus(c)
		if err != nil {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: " + err.Error())
		}

		if stat.Code() != 220 {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: unexpected response to STARTTLS: " + stat.String())
		}

		c = tls.Client(c, &tls.Config{ServerName: domain})
		err = c.(*tls.Conn).Handshake()
		if err != nil {
			c.Close()
			return nil, errors.New("connectStartTLS: handshake error: " + err.Error())
		}

		err = packets.NewCommand("EHLO", conf.Mxdomain).Send(c)
		if err != nil {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: post-handshake greeting error: " + err.Error())
		}

		stat, err = readAndParseStatus(c)
		if err != nil {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: " + err.Error())
		}

		if stat.Code() != 250 {
			packets.NewCommand("QUIT").Send(c)
			c.Close()
			return nil, errors.New("connectStartTLS: unexpected response to post-handshake greeting: " + stat.String())
		}
	}
	return c, nil
}

func readAndParseStatus(c net.Conn) (*packets.Status, error) {
	b, err := readResponse(c)
	if err != nil {
		return nil, errors.New("readAndParseStatus: error reading response: " + err.Error())
	}

	stat, err := packets.ParseStatus(b)
	if err != nil {
		return nil, errors.New("readAndParseStatus: error parsing status: " + err.Error())
	}

	return stat, nil
}
