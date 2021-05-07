package wang

import (
	"testing"
)

func TestLoc(t *testing.T) {
	l := loc{2, 4}
	if l.foff() != 9216 {
		t.Fatalf("expected 9216, got %d", l.foff())
	}

}

func TestLabel(t *testing.T) {
	l := [...]byte{0x03, 0x20, 0x75, 0x30, 0x33, 0x32, 0x30, 0x75}
	var tg tag
	copy(tg[:], l[:])
	if tg.String() != string(l[3:8]) {
		t.Fatal("label didn't verify")
	}
}

func TestDate(t *testing.T) {
	d := []byte{0x30, 0x35, 0x00,
		0x31, 0x38, 0x00,
		0x38, 0x39, 0x00,
		0x31, 0x36, 0x00,
		0x30, 0x34}
	ti := date(d)
	if ti.Format(timefmt) != "05.18.89 16:04" {
		t.Fatal(ti)
	}
}
