package textbuf

import (
	"errors"

	font "github.com/ajanata/oled_font"
	"tinygo.org/x/drivers"
)

type Buffer struct {
	dev        drivers.Displayer
	disp       font.Display
	buf        [][]byte
	width      int16
	height     int16
	fontWidth  int16
	fontHeight int16
	x          int16
	y          int16
}

type FontSize uint8

const (
	FontSize6x8   = font.FONT_6x8
	FontSize7x10  = font.FONT_7x10
	FontSize11x18 = font.FONT_11x18
	FontSize16x26 = font.FONT_16x26
)

func New(dev drivers.Displayer, size FontSize) (*Buffer, error) {
	var fw, fh int16
	switch size {
	case FontSize6x8:
		fw, fh = 6, 8
	case FontSize7x10:
		fw, fh = 7, 10
	case FontSize11x18:
		fw, fh = 11, 18
	case FontSize16x26:
		fw, fh = 16, 26
	default:
		return nil, errors.New("invalid font size")
	}

	sw, sh := dev.Size()

	w := sw / (fw + 1)
	h := sh / (fh + 1)

	buf := make([][]byte, h)
	for i := int16(0); i < h; i++ {
		buf[i] = make([]byte, w)
	}

	b := Buffer{
		dev:        dev,
		disp:       font.NewDisplay(dev),
		buf:        buf,
		width:      w,
		height:     h,
		fontWidth:  fw,
		fontHeight: fh,
	}
	b.disp.Configure(font.Config{FontType: uint8(size)})
	return &b, b.Clear()
}

func (b *Buffer) Display() error {
	b.disp.YPos = 0
	for i := range b.buf {
		b.disp.XPos = 0
		for j := range b.buf[i] {
			b.disp.PrintChar(b.buf[i][j])
			b.disp.XPos += b.fontWidth + 1
		}
		b.disp.YPos += b.fontHeight + 1
	}

	return b.dev.Display()
}

func (b *Buffer) Clear() error {
	b.x, b.y = 0, 0
	for i := range b.buf {
		for j := range b.buf[i] {
			b.buf[i][j] = ' '
		}
	}
	return b.Display()
}

// Scroll moves each line of the display up by one and blanks the last line.
func (b *Buffer) Scroll() error {
	for i := 1; i < len(b.buf); i++ {
		copy(b.buf[i-1], b.buf[i])
	}
	last := len(b.buf) - 1
	for i := range b.buf[last] {
		b.buf[last][i] = ' '
	}
	return b.Display()
}

// Size returns the number of columns and rows of text on the display.
func (b *Buffer) Size() (int16, int16) {
	return b.width, b.height
}

func sanitize(ch byte) byte {
	if ch < ' ' || ch > '~' {
		return '?'
	}
	return ch
}

func (b *Buffer) SetLine(line int16, text string) error {
	if line > b.height {
		return errors.New("not that many lines")
	}
	// TODO handle strings longer than the screen width?
	for i := 0; int16(i) < b.width; i++ {
		var ch byte = ' '
		if i < len(text) {
			ch = text[i]
		}
		b.buf[line][i] = sanitize(ch)
	}
	return b.Display()
}

func (b *Buffer) Println(text string) error {
	return b.Print(text + "\n")
}

func (b *Buffer) Print(text string) error {
	err := b.print(text)
	if err != nil {
		return err
	}
	return b.Display()
}

func (b *Buffer) print(text string) error {
	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch ch {
		case '\r':
			continue
		case '\n':
			b.x = 0
			if b.y == b.height-1 {
				err := b.Scroll()
				if err != nil {
					return err
				}
			} else {
				b.y++
			}
		case '\t':
			err := b.print("  ")
			if err != nil {
				return err
			}
		default:
			err := b.putc(ch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Buffer) putc(ch byte) error {
	b.buf[b.y][b.x] = sanitize(ch)
	b.x++
	if b.x == b.width {
		b.x = 0
		if b.y == b.height-1 {
			err := b.Scroll()
			if err != nil {
				return err
			}
		} else {
			b.y++
		}
	}
	return nil
}
