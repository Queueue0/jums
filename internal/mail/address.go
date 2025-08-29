package mail

import (
	"errors"
	"fmt"
	"strings"
)

type Address struct {
	User   string
	Domain string
}

func NewAddress(emailaddr string) (*Address, error) {
	parts := strings.Split(emailaddr, "@")
	if len(parts) != 2 {
		return nil, errors.New(fmt.Sprintf("%s is not a valid email address", emailaddr))
	}

	// convert domain to lower for easy grouping, but not username
	// it's up to the receiving mail server to determine if the username is case sensitive,
	// so we must treat it as if it is
	return &Address{parts[0], strings.ToLower(parts[1])}, nil
}

func (a *Address) String() string {
	return fmt.Sprintf("%s@%s", a.User, a.Domain)
}

func (a *Address) SmtpFormat() string {
	return fmt.Sprintf("<%s@%s>", a.User, a.Domain)
}
