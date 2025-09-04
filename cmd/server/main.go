package main

import (
	"crypto/tls"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/smtp"
)

func main() {
	conf := config.GetConfig()
	var loglevel slog.Level
	switch strings.ToUpper(conf.LogLevel) {
	case "DEBUG":
		loglevel = slog.LevelDebug
	case "INFO":
		loglevel = slog.LevelInfo
	case "WARN":
		loglevel = slog.LevelWarn
	case "ERROR":
		loglevel = slog.LevelError
	default:
		loglevel = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level: loglevel,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
	slog.Info("Josh's Unremarkable Mail Server started, listening for connections")

	go func() {
		srv25, err := net.Listen("tcp", ":25")
		if err != nil {
			slog.Error("Fatal error on port 25", "msg", err.Error())
			panic(err)
		}
		defer srv25.Close()
		for {
			c, err := srv25.Accept()
			if err != nil {
				c.Close()
				slog.Error("Error on port 25", "err", err.Error())
				continue
			}
			slog.Info("Received connection", "addr", c.RemoteAddr().String())

			go smtp.Handle(c)
		}
	}()

	go func() {
		srv587, err := net.Listen("tcp", ":587")
		if err != nil {
			slog.Error("Fatal error on port 587", "msg", err.Error())
			panic(err)
		}
		defer srv587.Close()
		for {
			c, err := srv587.Accept()
			if err != nil {
				c.Close()
				slog.Error("Error on port 587", "err", err.Error())
				continue
			}
			slog.Info("Received connection", "addr", c.RemoteAddr().String())

			go smtp.Handle(c)
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
		slog.Error("Fatal error on port 465", "msg", err.Error())
		panic(err)
	}
	defer srvTLS.Close()

	for {
		c, err := srvTLS.Accept()
		if err != nil {
			c.Close()
			slog.Error("Error on port 465", "err", err.Error())
			continue
		}
		slog.Info("Received connection", "addr", c.RemoteAddr().String())

		go smtp.Handle(c)
	}
}
