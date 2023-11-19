package wang

import (
	"io"
)

const (
	_           = iota
	centre             // 0x01 found at beginnings of lines to flag line centering
	tab                // horizontal tab character
	lineEnd            // produces <CR><LF> sequence. Terminates lines, indented blocks
	indent             // marks the start of an indented block
	dtab               // 0x05 "dec tab". Used in tables as column separator to enable right justification of numeric strings
	format             // marks beginning of a format line (to define tab stops). 0x20 tabs mark tab stops, 0x20 fill the line, 0x03 indicates right margin
	vline              // vertical line to separate table colums or for borders. Replaced with |
	degrees            // raised circle mark. Replaced with °
	noBreak            // non-breaking space, i.e. space with no line break
	pound              // 0x0A. Replaced with £
	stop               // not used
	note               // Used in tables, brackets bottom line of numeric value
	merge              // not used
	superscript        // raised values e.g. footnote
	subscript          // 0x0F ends block of superscript text
	page        = 0x86 // page break and beginning of a format block
	bold        = 0x8e // first occurrence turns it on, second turns it off
	uVline      = 0x87 // underlined vertical line |
	uDegrees    = 0x88 // underlined degrees °
	uNobreak    = 0x89 // underlined version of non-breaking space
)

type Token struct {
	Typ TokenType
	Off int64
	Val string
}

type TokenType int

const (
	TokenErr TokenType = iota
	TokenEOF
	TokenPage
	TokenFormat
	TokenText
	TokenUnderText // Underlined text
	TokenCentre
	TokenTab
	TokenEnd
	TokenIndent
	TokenDTab
	TokenNoBreak
	TokenNote
	TokenSuper
	TokenSub
	TokenBold
	TokenUnderNoBreak
)

func (t TokenType) String() string {
	switch t {
	case TokenErr:
		return "Error"
	case TokenEOF:
		return "End of File"
	case TokenPage:
		return "Page Break"
	case TokenFormat:
		return "Format Line"
	case TokenText:
		return "Text"
	case TokenUnderText:
		return "Underlined Text"
	case TokenCentre:
		return "Centred"
	case TokenTab:
		return "Tab"
	case TokenEnd:
		return "Line End"
	case TokenIndent:
		return "Indent"
	case TokenDTab:
		return "Right-justified Tab"
	case TokenNoBreak:
		return "Non-breaking space"
	case TokenNote:
		return "Note"
	case TokenSuper:
		return "Superscript"
	case TokenSub:
		return "Subscript"
	case TokenBold:
		return "Bold"
	case TokenUnderNoBreak:
		return "Underlined non-breaking space"
	default:
		return "Unknown Token"
	}
}

const bufSz int = 512

type Decoder struct {
	fIdx        int64 // index in the underlying reader
	readBuffer  [bufSz]byte
	rbuf        []byte
	writeBuffer [bufSz]byte
	wLen        int // length written to writeBuffer
	r           io.Reader
	eof         bool
	curr        state
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// reports if a character is underlined and, if so, decodes the character
func isUnder(char byte) (byte, bool) {
	if char == 0x86 || char == 0x8e {
		return char, false
	}
	if char&0x7F == char {
		return char, false
	}
	return char & 0x7F, true
}

type state int

const (
	ready state = iota
	inText
	inUnderline
	inPageFormat
	inFormat
)

// Only four accumulate: text, underlined text, page (format), format (format)

func (d *Decoder) Token() (Token, error) {
	if d.eof {
		return Token{Off: d.fIdx}, io.EOF
	}
	for {
		for _, c := range d.rbuf {
			switch c {
			case centre:
				return Token{Typ: TokenCentre}, nil
			}
			_ = c
			d.fIdx += 1
		}
		n, err := d.r.Read(d.readBuffer[:])
		if n < 1 {
			d.eof = true
			// do we have an open token to finalize first?
			return Token{Off: d.fIdx}, err
		}
		d.rbuf = d.readBuffer[:n]
	}
}

// Tage a page or format token and return indents and line length
func FormatToken(t Token) ([]int, int) {
	return nil, 0
}
