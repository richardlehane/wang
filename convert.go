package wang

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

type Token struct {
	Typ TokenType
	Off int64
	Val string
}

type TokenType int

const (
	TokenNull TokenType = iota
	TokenErr
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
	case TokenNote:
		return "Note"
	case TokenSuper:
		return "Superscript"
	case TokenSub:
		return "Subscript"
	case TokenBold:
		return "Bold"
	default:
		return "Unknown Token"
	}
}

const bufSz int = 4096

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
	return &Decoder{fIdx: -1, r: r}
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
	inFormat
	inSpecial
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
	noBreak            // non-breaking space, i.e. space with no line break. Replace with
	pound              // 0x0A. Replaced with £
	stop               // not used
	note               // Used in tables, brackets bottom line of numeric value
	merge              // not used
	superscript        // raised values e.g. footnote
	subscript          // 0x0F ends block of superscript text
	page        = 0x86 // page break and beginning of a format block
	bold        = 0x8e // first occurrence turns it on, second turns it off
)

func special(b byte, off int64) Token {
	switch b {
	case centre:
		return Token{Typ: TokenCentre, Off: off}
	case tab:
		return Token{Typ: TokenTab, Off: off, Val: "\t"}
	case lineEnd:
		return Token{Typ: TokenEnd, Off: off, Val: "\n"}
	case indent:
		return Token{Typ: TokenIndent, Off: off}
	case dtab:
		return Token{Typ: TokenDTab, Off: off, Val: "\t"}
	case note:
		return Token{Typ: TokenNote, Off: off}
	case superscript:
		return Token{Typ: TokenSuper, Off: off}
	case subscript:
		return Token{Typ: TokenSub, Off: off}
	case page:
		return Token{Typ: TokenPage, Off: off}
	case bold:
		return Token{Typ: TokenBold, Off: off}
	default:
		return Token{Typ: TokenErr, Off: off, Val: fmt.Sprintf("unknown special character %c", b)}
	}
}

func (d *Decoder) text() Token {
	typ := TokenText
	if d.curr == inUnderline {
		typ = TokenUnderText
	}
	off := d.fIdx - int64(utf8.RuneCountInString(string(d.writeBuffer[:d.wLen])))
	tok := Token{
		Typ: typ,
		Off: off,
		Val: string(string(d.writeBuffer[:d.wLen])),
	}
	d.wLen = 0
	return tok
}

func (d *Decoder) Token() (Token, error) {
	if d.eof {
		return Token{Typ: TokenEOF, Off: d.fIdx}, io.EOF
	}
	for {
		for idx, c := range d.rbuf {
			d.fIdx += 1
			// if we cached a page break then we need to change state to inFormat
			if d.curr == inSpecial && d.writeBuffer[0] == page {
				d.curr = inFormat
				tok := special(page, d.fIdx-1)
				d.writeBuffer[0] = c
				d.wLen = 1
				d.rbuf = d.rbuf[idx+1:]
				return tok, nil
			}
			// are we in a format line?
			if d.curr == inFormat {
				switch c {
				case 0x31, 0x20, 0x02:
					d.writeBuffer[d.wLen] = c
					d.wLen += 1
				case 0x03:
					d.curr = ready
					tok := Token{
						Typ: TokenFormat,
						Off: d.fIdx - int64(d.wLen) - 1,
						Val: string(d.writeBuffer[:d.wLen]),
					}
					d.wLen = 0
					d.rbuf = d.rbuf[idx+1:]
					return tok, nil
				default:
					d.curr = ready
					tok := Token{
						Typ: TokenErr,
						Off: d.fIdx,
						Val: fmt.Sprintf("bad format line character %c at offset %d", c, d.fIdx),
					}
					d.wLen = 0
					d.rbuf = d.rbuf[idx+1:]
					return tok, fmt.Errorf("bad format line character %c at offset %d", c, d.fIdx)
				}
				continue // keep looping if 0x31, 0x20 or 0x02
			}
			uc, under := isUnder(c)
			r := WangWorldLanguages(uc)
			var prev Token
			switch {
			case d.curr == inSpecial:
				prev = special(d.writeBuffer[0], d.fIdx-1)
				d.wLen = 0
			case (r == 0 && (d.curr == inText || d.curr == inUnderline)) || (d.curr == inText && under) || (d.curr == inUnderline && !under):
				prev = d.text()
			}
			if r > 0 { // text token
				d.wLen += utf8.EncodeRune(d.writeBuffer[d.wLen:], r)
				if under {
					d.curr = inUnderline
				} else {
					d.curr = inText
				}
				if prev.Typ > TokenNull {
					d.rbuf = d.rbuf[idx+1:]
					return prev, nil
				}
				continue // keep looping to consume more text
			}
			// we have a special token
			// 	if we need to cache...
			if prev.Typ > TokenNull {
				if c == format {
					d.curr = inFormat
				} else {
					d.writeBuffer[0] = c
					d.wLen = 1
					d.curr = inSpecial
				}
				d.rbuf = d.rbuf[idx+1:]
				return prev, nil
			}
			if c == format {
				d.curr = inFormat
				continue
			}
			if c == page {
				d.curr = inFormat
			} else {
				d.curr = ready
			}
			d.rbuf = d.rbuf[idx+1:]
			return special(c, d.fIdx), nil
		}
		n, err := d.r.Read(d.readBuffer[:])
		if n < 1 {
			d.eof = true
			switch d.curr {
			case inSpecial:
				return special(d.writeBuffer[0], d.fIdx-1), nil
			case inText, inUnderline:
				return d.text(), nil
			}
			if err == io.EOF {
				return Token{Typ: TokenEOF, Off: d.fIdx}, err
			}
			return Token{Typ: TokenErr, Off: d.fIdx, Val: err.Error()}, err
		}
		d.rbuf = d.readBuffer[:n]
	}
}

// Tage a page or format token and return line spacing, indents and line length
func FormatToken(t Token) (int, []int, int) {
	if t.Typ != TokenFormat || len(t.Val) < 2 {
		return 0, nil, 0
	}
	var spacing, length int
	if spc, err := strconv.Atoi(t.Val[:1]); err == nil {
		spacing = spc
	}
	tabs := make([]int, 0, 10)
	for _, c := range t.Val[1:] {
		switch c {
		case 0x20:
			length += 1
		case 0x02:
			tabs = append(tabs, length)
			length += 1
		}
	}
	return spacing, tabs, length
}

// WangWorldLanguages converts a character in the Wang World Lanaguages Character Set
// to a UTF-8 rune
func WangWorldLanguages(char byte) rune {
	switch {
	case (char >= 0x20 && char <= 0x5B) || char == 0x5D || (char >= 0x61 && char <= 0x7A): // 0x20 to 0x5B, 0x5D, 0x61 to 0x7A = ASCII
		return rune(char)
	case char >= 0x07 && char <= 0x0A:
		return specialChars[char-0x07]
	case char >= 0x10 && char <= 0x1F:
		return lowerChars[char-0x10]
	case char >= 0x5C && char <= 0x60:
		return midChars[char-0x5C]
	case char >= 0x7B && char <= 0x7F:
		return upperChars[char-0x7B]
	default:
		return 0
	}
}

var specialChars = [4]rune{ //0x07 to 0x0A
	'|', '°', 0xA0, '£',
}

var lowerChars = [16]rune{ // 0x10 to 0x1F
	'â', 'ê', 'î', 'ô', 'û',
	'ä', 'ë', 'ï', 'ö', 'ü',
	'à', 'è', 'ì', 'ò', 'ù',
	'ç',
}

var midChars = [5]rune{ //0x5C, 0x5E to 0x60
	'Ñ', 0, 'ñ', '¿', '¡',
}

var upperChars = [5]rune{ // 0x7B to 0x7F
	'á', 'é', 'í', 'ó', 'ú',
}

type EncodeFn func(*Decoder, io.Writer) error

func TextEncode(dec *Decoder, w io.Writer) error {
	buf := &bytes.Buffer{}
	for {
		tok, err := dec.Token()
		if err == io.EOF || tok.Typ == TokenEOF {
			_, err = buf.WriteTo(w)
			return err
		}
		switch tok.Typ {
		case TokenText, TokenUnderText, TokenTab, TokenEnd, TokenDTab:
			_, err = buf.WriteString(tok.Val)
			if err != nil {
				return err
			}
		}
	}
}

var repl = strings.NewReplacer("\\", "\\\\", "{", "\\{", "}", "\\}")

func ansi(in string) string {
	out, _ := charmap.Windows1252.NewEncoder().String(in)
	out = repl.Replace(out)
	var hasSpecial bool
	for _, c := range []byte(out) {
		if c > 126 {
			hasSpecial = true
		}
	}
	if !hasSpecial {
		return out
	}
	buf := make([]byte, 0, len(out))
	for _, c := range []byte(out) {
		if c > 126 {
			hx := "\\'" + hex.EncodeToString([]byte{c})
			buf = append(buf, []byte(hx)...)
		} else {
			buf = append(buf, c)
		}
	}
	return string(buf)
}

func RTFEncode(dec *Decoder, w io.Writer) error {
	var inBold bool
	buf := &bytes.Buffer{}
	buf.WriteString(`{\rtf1\ansi\deff0 {\fonttbl {\f0\fmodern Courier New;}}`)
	for {
		tok, err := dec.Token()
		if err == io.EOF || tok.Typ == TokenEOF {
			buf.WriteString("}")
			_, err = buf.WriteTo(w)
			return err
		}
		switch tok.Typ {
		case TokenBold:
			if inBold {
				inBold = false
				buf.WriteString("}")
			} else {
				inBold = true
				buf.WriteString("{\\b ")
			}
		case TokenEnd:
			buf.WriteString("\\line ")
		case TokenText, TokenUnderText:
			buf.WriteString(`\f0\fs24 `)
			_, err = buf.WriteString(ansi(tok.Val))
			if err != nil {
				return err
			}
		}
	}
}
