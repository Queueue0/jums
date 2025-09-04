package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/Queueue0/jums/internal/config"
	"github.com/Queueue0/jums/internal/smtp/packets"
)

type Mail struct {
	From     *Address
	Rcpt     []Address
	Data     []byte
	Id       string
	Received PartialReceived
}

func (m *Mail) Send() error {
	fmt.Println("Sending mail...")
	var ce *CompiledErrors = nil
	appendErr := func(err error) {
		if ce == nil {
			ce = &CompiledErrors{}
		}
		ce.Append(err)
	}

	g := m.groupRcpts()
	config := config.GetConfig()
	for domain, addrs := range g {
		if domain == config.Domain {
			for _, addr := range addrs {
				m.Deliver(addr)
			}
			continue
		}

		c, err := createSMTPConn(domain)
		if err != nil {
			appendErr(err)
			continue
		}

		for _, addr := range addrs {
			fmt.Println(m.Received.format(addr))
			data := append([]byte(m.Received.format(addr)), m.Data...)
			// TODO: handle these errors
			_ = packets.NewCommand("MAIL", fmt.Sprintf("FROM:<%s>", m.From.String())).Send(c)
			s, _ := readAndParseStatus(c)
			if s.Code() != 250 {
				_ = packets.NewCommand("RSET").Send(c)
				readAndParseStatus(c)
				continue
			}

			_ = packets.NewCommand("RCPT", fmt.Sprintf("TO:<%s>", addr.String())).Send(c)
			s, _ = readAndParseStatus(c)
			if s.Code() != 250 {
				_ = packets.NewCommand("RSET").Send(c)
				readAndParseStatus(c)
				continue
			}

			_ = packets.NewCommand("DATA").Send(c)
			s, _ = readAndParseStatus(c)
			// accept 250 even though it's not strictly in spec
			if s.Code() != 354 && s.Code() != 250 {
				_ = packets.NewCommand("RSET").Send(c)
				readAndParseStatus(c)
				continue
			}

			c.Write(data)
			readAndParseStatus(c)
		}
		_ = packets.NewCommand("QUIT").Send(c)
		readAndParseStatus(c)
		c.Close()
	}

	return ce
}

// Deliver mail addressed to users at our domain
func (m *Mail) Deliver(addr Address) error {
	return nil
}

func (m *Mail) GenerateId() error {
	h := sha256.New()
	_, err := h.Write(m.Data)
	if err != nil {
		return err
	}

	m.Id = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (m *Mail) groupRcpts() map[string][]Address {
	g := make(map[string][]Address)
	for _, r := range m.Rcpt {
		if _, ok := g[r.Domain]; !ok {
			g[r.Domain] = []Address{}
		}
		g[r.Domain] = append(g[r.Domain], r)
	}

	return g
}

type PartialReceived struct {
	From      string
	By        string
	With      string
	TlsInfo   string
	Id        string
	Timestamp string
}

func (r *PartialReceived) format(forAddr Address) string {
	return fmt.Sprintf("Received: %s\r\n\t%s %s\r\n\t%s\r\n\t%s\r\n\tfor %s;\r\n\t%s\r\n", r.From, r.By, r.With, r.TlsInfo, r.Id, forAddr.SmtpFormat(), r.Timestamp)
}
