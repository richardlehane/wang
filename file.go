package wang

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type File struct {
	DocID     tag
	ArchiveID string
	Name      string
	Comment   string
	Author    string
	Operator  string
	Created   time.Time
	Modified  time.Time
	pages     []loc
	pgMap     map[loc]struct{}
	pgIdx     int // page index
	sect      loc // current sector
	secIdx    int // index within sector
	r         *Reader
}

func (f *File) String() string {
	return fmt.Sprintf(
		"Document ID:   %s\n"+
			"Archive ID:    %s\n"+
			"Document Name: %s\n"+
			"Author:        %s\n"+
			"Operator:      %s\n"+
			"Comments:      %s\n"+
			"Created:       %s\n"+
			"Modified:      %s\n",
		f.DocID.String(), f.ArchiveID, f.Name, f.Author, f.Operator, f.Comment, f.Created, f.Modified)
}

var sanitizer = strings.NewReplacer(`/`, "_", `\`, "_", `?`, "_", `%`, "_", `*`, "_", `:`, "_", `|`, "_", `"`, "_", `<`, "_", `>`, "_")

func (f *File) SanitizedName() string {
	return sanitizer.Replace(f.Name)
}

func (f *File) CSV() []string {
	return []string{f.DocID.String(), f.ArchiveID, f.Name, f.Author, f.Operator, f.Comment, f.Created.String(), f.Modified.String()}
}

func file(buf []byte) (*File, loc, error) {
	if len(buf) < secsz {
		return nil, loc{}, fmt.Errorf("sector not big enough for file metadata: %d", len(buf))
	}
	f := &File{
		Name:      WWLString(trim(buf[13:38])),
		ArchiveID: trim(buf[192:197]),
		Author:    WWLString(trim(buf[60:80])),
		Operator:  WWLString(trim(buf[39:59])),
		Comment:   WWLString(trim(buf[81:101])),
		Created:   date(buf[132:146]),
		Modified:  date(buf[177:191]),
	}
	copy(f.DocID[:], buf[4:7])
	return f, loc{buf[0], buf[1]}, nil
}

func trim(buf []byte) string {
	return strings.TrimSpace(string(buf))
}

func date(buf []byte) time.Time {
	//mmddyyhhmm
	str := fmt.Sprintf("%s.%s.%s %s:%s",
		string(buf[:2]),
		string(buf[3:5]),
		string(buf[6:8]),
		string(buf[9:11]),
		string(buf[12:14]))
	t, _ := time.Parse(timefmt, str)
	return t
}

func pages(buf []byte) []loc {
	if len(buf) < 256 {
		return nil
	}
	num := int(buf[2])
	pgs := make([]loc, num)
	for i := 0; i < num; i++ {
		copy(pgs[i][:], buf[16+i*2:18+i*2])
	}
	return pgs
}

// Read implements the io.Reader interface
func (f *File) Read(b []byte) (int, error) {
	var idx, n int
	if f.pgIdx >= len(f.pages) {
		return n, io.EOF
	}
	ploc := f.pages[f.pgIdx]
	if f.sect.zero() {
		f.sect = ploc
	}
	for {
		for {
			byt, err := f.r.sector(f.sect)
			if err != nil {
				return n, err
			}
			var nxt loc
			copy(nxt[:], byt) // take the next location
			length := int(byt[2])
			if length > 255 {
				return n, errors.New("bad length")
			}
			byt = byt[7 : length+1]
			byt = byt[f.secIdx:]
			rem := len(b) - idx
			if rem < 1 {
				return n, nil
			}
			cp := copy(b[idx:], byt)
			n += cp
			idx += cp
			if cp < len(byt) {
				f.secIdx = cp
				return n, nil
			}
			f.secIdx = 0
			f.sect = nxt
			if _, ok := f.pgMap[f.sect]; ok { // if the next location is one of our page locations, we're done for the current page
				break
			}
		}
		f.pgIdx += 1
		if f.pgIdx < len(f.pages) { // if we have more pages, increment the page and sector
			ploc = f.pages[f.pgIdx]
			f.sect = ploc
		} else {
			break
		}
	}
	return n, io.EOF
}
