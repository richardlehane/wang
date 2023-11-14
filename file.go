package wang

import (
	"fmt"
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

func file(buf []byte) (*File, loc, error) {
	if len(buf) < secsz {
		return nil, loc{}, fmt.Errorf("sector not big enough for file metadata: %d", len(buf))
	}
	f := &File{
		Name:      trim(buf[13:38]),
		ArchiveID: trim(buf[192:196]),
		Author:    trim(buf[60:80]),
		Operator:  trim(buf[39:59]),
		Comment:   trim(buf[81:101]),
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
