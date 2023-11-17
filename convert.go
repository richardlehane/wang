package wang

import "io"

const (
	_           = iota
	centre             // 0x01 found at beginnings of lines to flag line centering
	tab                // horizontal tab character
	lineEnd            // produces <CR><LF> sequence
	indent             // marks the start of an indented block
	dtab               // 0x05 "dec tab". Used in tables as column separator to enable right justification of numeric strings
	format             // marks beginning of a format line (to define tab stops). 0x20 tabs mark tab stops, 0x20 fill the line, 0x03 indicates right margin
	vline              // vertical line to separate table colums or for borders
	degrees            // raised circle mark
	noBreak            // non-breaking space, i.e. space with no line break
	pound              // 0x0A
	stop               // not used
	note               // Used in tables, brackets bottom line of numeric value
	merge              // not used
	superscript        // raised values e.g. footnote
	subscript          // 0x0F ends block of superscript text
	page        = 0x86 // page break and beginning of a format block
	bold        = 0x8e // first occurrence turns it on, second turns it off
)

type Decoder struct{}

func NewDecoder(r io.Reader) *Decoder {
	return nil
}
