package packets

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Status struct {
	code  uint16
	lines []string
}

func (s *Status) Code() uint16 {
	return s.code
}

func (s *Status) String() string {
	out := ""
	for i, l := range s.lines {
		if i == len(s.lines)-1 {
			out = fmt.Sprintf("%s%d %s\r\n", out, s.code, l)
		} else {
			out = fmt.Sprintf("%s%d-%s\r\n", out, s.code, l)
		}
	}

	return out
}

func (s *Status) Bytes() []byte {
	return []byte(s.String())
}

func (s *Status) Send(c net.Conn) error {
	_, err := c.Write(s.Bytes())
	return err
}

func ParseStatus(b []byte) (*Status, error) {
	code, err := strconv.Atoi(string(b[:3]))
	if err != nil {
		return nil, err
	}

	bstr := strings.TrimSpace(string(b))
	var stat *Status
	lines := strings.Split(bstr, "\r\n")
	for i := range lines {
		if len(lines[i]) > 4 {
			lines[i] = lines[i][4:]
		}
	}
	stat = NewStatus(uint16(code), lines...)

	return stat, nil
}

func NewStatus(code uint16, lines ...string) (*Status) {
	return &Status{code, lines}
}
