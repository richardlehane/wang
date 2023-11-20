package wang

import (
	"os"
	"testing"
)

func TestLoc(t *testing.T) {
	l := loc{2, 4}
	if l.foff() != 9216 {
		t.Fatalf("expected 9216, got %d", l.foff())
	}

}

func TestOffToLoc(t *testing.T) {
	l := loc{2, 4}
	off := l.foff()
	nl := offToLoc(off)
	if l != nl {
		t.Fatalf("expected equality got %v and %v", l, nl)
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

func TestFile(t *testing.T) {
	_ = os.RemoveAll("examples/DAR-0015/files")
	if err := os.MkdirAll("examples/DAR-0015/files", 0777); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("examples/DAR-0015-001.img")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rdr, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	err = rdr.DumpFiles("examples/DAR-0015/files")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSectors(t *testing.T) {
	_ = os.RemoveAll("examples/DAR-0015/sectors")
	if err := os.Mkdir("examples/DAR-0015/sectors", 0777); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("examples/DAR-0015-001.img")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rdr, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	err = rdr.DumpSectors("examples/DAR-0015/sectors")
	if err != nil {
		t.Fatal(err)
	}
}

func TestText(t *testing.T) {
	_ = os.RemoveAll("examples/DAR-0015/text")
	if err := os.Mkdir("examples/DAR-0015/text", 0777); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("examples/DAR-0015-001.img")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rdr, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	err = rdr.DumpText("examples/DAR-0015/text")
	if err != nil {
		t.Fatal(err)
	}
}
