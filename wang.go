package wang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	secsz   = 256
	blocksz = secsz * 8
	chunksz = blocksz * 2
	diroff  = secsz * 3

	timefmt = "01.02.06 15:04"
)

func New(ra io.ReaderAt) (*Reader, error) {
	rdr := &Reader{ra: ra}
	err := rdr.loadChunk(0)
	if err != nil {
		return nil, err
	}
	if rdr.csz < blocksz {
		return nil, fmt.Errorf("file too small to be a wang, expect %d, got %d", blocksz, rdr.csz)
	}
	copy(rdr.archiveID[:], rdr.chunk[:3])
	// check archiveID
	if rdr.archiveID.String() != string(rdr.chunk[3:8]) {
		return nil, fmt.Errorf("bad wang, label did not verify")
	}
	// load the directory (at offset 768)
	rdr.contents = make([]dir, 0, 10)
	for i := 0; i+1 < secsz; i = i + 6 {
		t := tag{}
		l := loc{}
		copy(t[:], rdr.chunk[diroff+i:])
		copy(l[:], rdr.chunk[diroff+i+3:])
		if t.zero() && l.zero() {
			break
		}
		rdr.contents = append(rdr.contents, dir{t, l})
	}
	rdr.Files = make([]*File, len(rdr.contents))
	for idx, d := range rdr.contents {
		buf, err := rdr.sector(d.l)
		if err != nil {
			return nil, err
		}
		f, pgl, err := file(buf)
		if err != nil {
			return nil, err
		}
		buf, err = rdr.sector(pgl)
		if err != nil {
			return nil, err
		}
		f.pages = pages(buf)
		rdr.Files[idx] = f
	}
	return rdr, nil
}

// Reader provides sequential access to a wang img
type Reader struct {
	archiveID tag
	contents  []dir
	Files     []*File

	chunk [chunksz]byte
	coff  int64
	csz   int
	ra    io.ReaderAt
}

func (r *Reader) loadChunk(i int64) error {
	if r.coff == i && r.csz > 0 {
		return nil
	}
	sz, err := r.ra.ReadAt(r.chunk[:], i)
	if err != nil {
		return err
	}
	r.csz = sz
	r.coff = i
	return nil
}

func (r *Reader) sector(l loc) ([]byte, error) {
	if err := r.loadChunk(l.chunkoff()); err != nil {
		return nil, err
	}
	if l.secoff()+secsz > r.csz {
		return nil, fmt.Errorf("can't seek that far: %v", l)
	}
	return r.chunk[l.secoff() : l.secoff()+secsz], nil
}

func (r *Reader) DumpSectors() error {
	for idx, d := range r.contents {
		fld := fmt.Sprintf("%02d", idx)
		err := os.Mkdir(fld, 0777)
		if err != nil && !os.IsExist(err) {
			return err
		}
		var jidx int
		l := d.l
		for !l.zero() {
			byt, err := r.sector(l)
			if err != nil {
				return err
			}
			err = os.WriteFile(filepath.Join(fld, fmt.Sprintf("%02d.dmp", jidx)), byt, 0777)
			if err != nil {
				return err
			}
			jidx++
			copy(l[:], byt)
		}
	}
	return nil
}

func (r *Reader) DumpFiles(path string) error {
	for _, f := range r.Files {
		buf := &bytes.Buffer{}
		for _, l := range f.pages {
			for {
				byt, err := r.sector(l)
				if err != nil {
					return err
				}
				if len(byt) < 7 {
					return errors.New("bad sector")
				}
				copy(l[:], byt) // take the next location
				length := int(byt[2])
				if length > 255 {
					return errors.New("bad length")
				}
				byt = byt[7 : length+1]
				_, err = buf.Write(byt)
				if err != nil {
					return err
				}
				if length < 255 {
					break
				}
			}
		}
		err := os.WriteFile(filepath.Join(path, f.Name), buf.Bytes(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reader) Scan() map[tag][3][]int64 {
	llist := make(map[tag][]int64)
	for _, d := range r.contents {
		l := d.l
		for !l.zero() {
			llist[d.t] = append(llist[d.t], l.foff())
			byt, err := r.sector(l)
			if err != nil {
				break
			}
			copy(l[:], byt)
		}
	}
	llist2 := make(map[tag][]int64)
	for k, v := range llist {
		if len(v) < 2 {
			continue
		}
		byt, err := r.sector(offToLoc(v[1]))
		if err != nil {
			continue
		}
		for i := 8; i < len(byt)-2; i = i + 2 {
			nl := loc{}
			copy(nl[:], byt[i:])
			if !nl.zero() {
				llist2[k] = append(llist2[k], nl.foff())
			}
		}
	}
	sectors := make(map[tag][]int64)
	l := loc{0, 8}
	for {
		byt, err := r.sector(l)
		if err != nil {
			break
		}
		var t tag
		copy(t[:], byt[4:7])
		if !t.zero() {
			sectors[t] = append(sectors[t], l.foff())
		}
		l = l.inc()
	}
	ret := make(map[tag][3][]int64)
	for k, v := range llist {
		ret[k] = [3][]int64{v, llist2[k], sectors[k]}
	}
	return ret
}

type loc [2]byte

func offToLoc(i int64) loc {
	h := i / 4096
	m := i % 4096
	l := m / 256
	return loc{byte(h), byte(l)}
}

func (l loc) zero() bool {
	if l[0] == 0 && l[1] == 0 {
		return true
	}
	return false
}

func (l loc) inc() loc {
	if l[1] < 15 {
		l[1] = l[1] + 1
	} else {
		l[1] = 0
		l[0] = l[0] + 1
	}
	return l
}

func (l loc) chunkoff() int64 {
	return int64(l[0]) * chunksz
}

func (l loc) secoff() int {
	return int(l[1]) * secsz
}

func (l loc) foff() int64 {
	return l.chunkoff() + int64(l.secoff())
}

type tag [3]byte

func (t tag) String() string {
	return fmt.Sprintf("%02x%02x%c", t[0], t[1], t[2])
}

func (t tag) zero() bool {
	if t[0] == 0 && t[1] == 0 && t[2] == 0 {
		return true
	}
	return false
}

type dir struct {
	t tag
	l loc
	// plus one byte of padding
}

type header struct {
	next      loc
	sz        byte
	something byte
	t         tag
}

func trunc(buf []byte) []byte {
	if len(buf) < 3 {
		return []byte{}
	}
	len := int(buf[3])
	if len < 8 {
		return []byte{}
	}
	return buf[7:len]
}
