package wang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const (
	secsz   = 256        // the wang disks store date in 256 byte sectors
	chunksz = secsz * 16 // 4096 - the wang disks use 4096 byte chunks in their locations
	diroff  = secsz * 3

	timefmt = "01.02.06 15:04"
)

func New(ra io.ReaderAt) (*Reader, error) {
	rdr := &Reader{ra: ra}
	err := rdr.loadChunk(0)
	if err != nil {
		return nil, err
	}
	if rdr.csz < secsz*8 {
		return nil, errors.New("file too small to be a wang, need at least 2048 bytes")
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
		f.pgMap = make(map[loc]struct{})
		f.pgMap[loc{}] = struct{}{} // include the empty location (for last page in a file)
		for _, l := range f.pages {
			f.pgMap[l] = struct{}{}
		}
		f.r = rdr
		rdr.Files[idx] = f
	}
	return rdr, nil
}

// Reader provides sequential access to a Wang disk image
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

// Sector returns a 256 byte slice for the sector at location l
func (r *Reader) sector(l loc) ([]byte, error) {
	if err := r.loadChunk(l.chunkoff()); err != nil {
		return nil, err
	}
	if l.secoff()+secsz > r.csz {
		return nil, fmt.Errorf("can't seek that far: %v", l)
	}
	return r.chunk[l.secoff() : l.secoff()+secsz], nil
}

// DumpSectors checks all 256 byte sectors in the file for tags
// Then dumps all 256 byte sectors for each tag.
func (r *Reader) DumpSectors(path string) error {
	start := loc{0, 4}
	var err error
	var byt []byte
	smap := make(map[tag][]loc)
	for byt, err = r.sector(start); err == nil; byt, err = r.sector(start) {
		var t tag
		copy(t[:], byt[4:7])
		if t.zero() {
			start = start.inc()
			continue
		}
		smap[t] = append(smap[t], start)
		start = start.inc()
	}
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return err
	}
	for k, v := range smap {
		_ = os.MkdirAll(filepath.Join(path, k.String()), 0777)
		for _, l := range v {
			byt, err := r.sector(l)
			if err != nil && err != io.EOF {
				return err
			}
			_ = os.WriteFile(filepath.Join(path, k.String(), strconv.Itoa(int(l.foff()))), byt, 0777)
		}
	}
	return err
}

// DumpFiles writes all files in the Wang disk to the path directory
func (r *Reader) DumpFiles(path string) error {
	for _, f := range r.Files {
		buf, err := io.ReadAll(f)
		if err == nil {
			err = os.WriteFile(filepath.Join(path, f.Name), buf, 0777)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reader) DumpEncoded(path string, ext string, fn EncodeFn) error {
	for _, f := range r.Files {
		dec := NewDecoder(f)
		buf := &bytes.Buffer{}
		err := fn(dec, buf)
		if err == nil {
			err = os.WriteFile(filepath.Join(path, f.Name+ext), buf.Bytes(), 0777)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reader) DumpText(path string) error {
	return r.DumpEncoded(path, ".txt", TextEncode)
}

func (r *Reader) DumpRTF(path string) error {
	return r.DumpEncoded(path, ".rtf", RTFEncode)
}

// Locations in Wang disks are stored in two bytes
// To calculate the file offset multiply the first byte by 4096 and the second byte by 256
type loc [2]byte

func offToLoc(i int64) loc {
	h := i / 4096
	m := i % 4096
	l := m / 256
	return loc{byte(h), byte(l)}
}

// is the location empty
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

// Tags are 3 byte unique identifiers for files.
// They are in the file directory at the start of a Wang disk and also present at the start of each content sector.
type tag [3]byte

func (t tag) String() string {
	return fmt.Sprintf("%02x%02x%c", t[0], t[1], t[2])
}

// empty sectors may be filled with 0 or 0xF6
func (t tag) zero() bool {
	if (t[0] == 0 || t[0] == 0xF6) && (t[1] == 0 || t[1] == 0xF6) && (t[2] == 0 || t[2] == 0xF6) {
		return true
	}
	return false
}

// directory entries start at 768 bytes
type dir struct {
	t tag
	l loc
	// plus one byte of padding
}
