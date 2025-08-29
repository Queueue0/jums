package mail

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidAddress = errors.New("invalid address")
)

type CompiledErrors struct {
	Errs []error
}

func (ce *CompiledErrors) Error() string {
	errStr := "Compiled Errors:\n"
	for i, e := range ce.Errs {
		errStr += fmt.Sprintf("\t%s", e.Error())
		if i != len(ce.Errs) - 1 {
			errStr += "\n"
		}
	}

	return errStr
}

func (ce *CompiledErrors) Append(errs ...error) {
	if ce.Errs == nil {
		ce.Errs = []error{}
	}
	ce.Errs = append(ce.Errs, errs...)
}
