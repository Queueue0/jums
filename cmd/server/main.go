package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/packets"
)

func main() {
	conf := config.GetConfig()
	fmt.Printf("Josh's Unremarkable Mail Server started, listening for connections to %s...\n", conf.Mxdomain)

	go func() {
		srv25, err := net.Listen("tcp", ":25")
		if err != nil {
			panic(err)
		}
		defer srv25.Close()
		for {
			c, err := srv25.Accept()
			if err != nil {
				c.Close()
				fmt.Fprintf(os.Stderr, "Error on port 25: %s", err.Error())
				continue
			}

			go handle(c)
		}
	}()

	go func() {
		srv587, err := net.Listen("tcp", ":587")
		if err != nil {
			panic(err)
		}
		defer srv587.Close()
		for {
			c, err := srv587.Accept()
			if err != nil {
				c.Close()
				fmt.Fprintf(os.Stderr, "Error on port 587: %s", err.Error())
				continue
			}

			go handle(c)
		}
	}()

	srvTLS, err := tls.Listen("tcp", ":465", &tls.Config{
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
	if err != nil {
		panic(err)
	}
	defer srvTLS.Close()

	for {
		c, err := srvTLS.Accept()
		if err != nil {
			c.Close()
			fmt.Fprintf(os.Stderr, "Error on port 465: %s", err.Error())
			continue
		}

		go handle(c)
	}
}

func handle(c net.Conn) {
	defer c.Close()
	fmt.Printf("Received connection from %s\n", c.RemoteAddr().String())
	respond(c, "220 Josh's Unremarkable Mail Server v0.0.0")
	state := newState(c)
	var reply *packets.Status
	for {
		//TODO: handle error
		b, err := readLine(state.Conn())
		if err != nil {
			fmt.Println(err.Error())
			state.Conn().Write(packets.NewStatus(421, "Unknown error").Bytes())
			break
		}
		state, reply = state.Handle(b)
		if reply != nil {
			reply.Send(state.Conn())
			if reply.Code()%100 == 21 {
				break
			}
		}
	}
	// We may have a completely different connection by the end, better safe than sorry
	state.Conn().Close()
}

func respond(c net.Conn, s string) {
	msg := s + "\r\n"
	c.Write([]byte(msg))
	fmt.Println(s)
}

// TODO: maxlength parameter, timeout
func readLine(c net.Conn) ([]byte, error) {
	read := []byte{}
	for len(read) < 2 || string(read[len(read)-2:]) != "\r\n" {
		next := make([]byte, 1)
		_, err := c.Read(next)
		if err != nil {
			return nil, err
		}
		read = append(read, next...)
	}

	fmt.Print(string(read))
	return read, nil
}

