package packets

import (
	"fmt"
	"net"
	"strings"
)

type Command struct {
	cmd  string
	args []string
}

func ParseCommand(b []byte) *Command {
	line := string(b)
	line = strings.TrimSpace(line)
	split := strings.Split(line, " ")

	return &Command{strings.ToUpper(split[0]), split[1:]}
}

func NewCommand(cmd string, args ...string) *Command {
	return &Command{cmd, args}
}

func (cmd *Command) Cmd() string {
	return cmd.cmd
}

func (cmd *Command) Args() []string {
	return cmd.args
}

func (cmd *Command) String() string {
	if len(cmd.args) > 0 {
		return strings.Join(append([]string{cmd.cmd}, cmd.args...), " ")
	}
	return cmd.cmd
}

func (cmd *Command) SafeString() string {
	return cmd.String()
}

func (cmd *Command) Bytes() []byte {
	return []byte(cmd.String() + "\r\n")
}

func (cmd *Command) Send(c net.Conn) error {
	fmt.Println(cmd.String())
	_, err := c.Write(cmd.Bytes())
	return err
}
