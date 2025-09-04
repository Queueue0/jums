package packets

import "testing"

func TestParseStatusMultiLine(t *testing.T) {
	mls := NewStatus(250, "Hi!", "This is a test", "You should be able to recreate this")

	out, err := ParseStatus(mls.Bytes())
	if err != nil {
		t.Error(err.Error())
		return
	}

	if out.Code() != mls.Code() {
		t.Errorf("out.Code() = %d, expected %d", out.Code(), mls.Code())
		return
	}

	if out.String() != mls.String() {
		t.Errorf("out.String() != mls.String()\nout.String():\n%s\nmls.String():\n%s", out.String(), mls.String())
	}
}

func TestParseStatusSingleLine(t *testing.T) {
	s := NewStatus(250, "Hi there!")

	out, err := ParseStatus(s.Bytes())
	if err != nil {
		t.Error(err.Error())
		return
	}

	if out.Code() != s.Code() {
		t.Errorf("out.Code() = %d, expected %d", out.Code(), s.Code())
		return
	}

	if out.String() != s.String() {
		t.Errorf("out.String() = \"%s\", expected \"%s\"", out.String(), s.String())
	}
}
