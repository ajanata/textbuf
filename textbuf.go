package textbuf

import (
	"errors"

	font "github.com/ajanata/oled_font"
	"tinygo.org/x/drivers"
)

// TODO make ...text versions of the print funcs to allow callers to avoid string concatenation (which allocates)

type Buffer struct {
	AutoFlush bool
	dev       drivers.Displayer
	disp      font.Display
	// high bit set = inverse video
	buf        [][]byte
	drawAll    bool
	draw       []bool
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

const inverseMask uint8 = 0b1000_0000

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

	w := sw / fw
	h := sh / fh

	buf := make([][]byte, h)
	for i := int16(0); i < h; i++ {
		buf[i] = make([]byte, w)
	}

	b := Buffer{
		dev:        dev,
		disp:       font.NewDisplay(dev),
		draw:       make([]bool, h),
		buf:        buf,
		width:      w,
		height:     h,
		fontWidth:  fw,
		fontHeight: fh,
	}
	b.disp.Configure(font.Config{FontType: uint8(size)})
	b.Clear()
	return &b, b.Display()
}

func (b *Buffer) drawLine(line int16) {
	b.disp.YPos = line * b.fontHeight
	b.disp.XPos = 0
	for i := range b.buf[line] {
		ch := b.buf[line][i]
		inverse := ch&inverseMask == inverseMask
		ch &^= inverseMask
		b.disp.PrintCharEx(ch, inverse)
		b.disp.XPos += b.fontWidth
	}
	b.disp.YPos += b.fontHeight
}

func (b *Buffer) Display() error {
	b.disp.YPos = 0
	update := false
	for i := range b.buf {
		if b.drawAll || b.draw[i] {
			b.draw[i] = false
			b.drawLine(int16(i))
			update = true
		}
	}
	b.drawAll = false

	if update {
		return b.dev.Display()
	} else {
		return nil
	}
}

func (b *Buffer) Clear() {
	b.drawAll = true
	b.x, b.y = 0, 0
	for i := range b.buf {
		for j := range b.buf[i] {
			b.buf[i][j] = ' '
		}
	}
}

// Scroll moves each line of the display up by one and blanks the last line.
func (b *Buffer) Scroll() {
	b.drawAll = true
	for i := 1; i < len(b.buf); i++ {
		copy(b.buf[i-1], b.buf[i])
	}
	last := len(b.buf) - 1
	for i := range b.buf[last] {
		b.buf[last][i] = ' '
	}
}

// Size returns the number of columns and rows of text on the display.
func (b *Buffer) Size() (int16, int16) {
	return b.width, b.height
}

func (b *Buffer) SetLine(line int16, text ...string) error {
	return b.setLine(line, false, text...)
}

func (b *Buffer) SetLineInverse(line int16, text ...string) error {
	return b.setLine(line, true, text...)
}

func (b *Buffer) setLine(line int16, inverse bool, text ...string) error {
	if line > b.height {
		return errors.New("not that many lines")
	}

	for i := 0; int16(i) < b.width; i++ {
		b.buf[line][i] = ' '
	}
	si := 0
	sl := 0
	// TODO handle strings longer than the screen width?
	for i := 0; int16(i) < b.width; i++ {
		var ch byte = ' '

	next:
		if si < len(text) {
			if i < len(text[si])+sl {
				ch = text[si][i-sl]
			} else {
				sl += len(text[si])
				si++
				goto next
			}
		}
		if inverse {
			ch |= inverseMask
		}
		b.buf[line][i] = ch
	}
	b.draw[line] = true
	if b.AutoFlush {
		return b.Display()
	}
	return nil
}

func (b *Buffer) Println(text string) error {
	return b.Print(text + "\n")
}

func (b *Buffer) PrintlnInverse(text string) error {
	return b.PrintInverse(text + "\n")
}

func (b *Buffer) Print(text string) error {
	return b.print(text, false)
}

func (b *Buffer) PrintInverse(text string) error {
	return b.print(text, true)
}

func (b *Buffer) print(text string, inverse bool) error {
	for i := 0; i < len(text); i++ {
		ch := text[i]
		switch ch {
		case '\r':
			continue
		case '\n':
			// make sure we didn't automatically line-wrap with the last character
			if !(i > 0 && b.x == 0) {
				b.x = 0
				if b.y == b.height-1 {
					b.Scroll()
				} else {
					b.y++
				}
			}
		case '\t':
			err := b.print("  ", inverse)
			if err != nil {
				return err
			}
		default:
			err := b.putc(ch, inverse)
			if err != nil {
				return err
			}
		}
	}
	if b.AutoFlush {
		return b.Display()
	}
	return nil
}

func (b *Buffer) putc(ch byte, inverse bool) error {
	if inverse {
		ch |= inverseMask
	}
	b.buf[b.y][b.x] = ch
	b.draw[b.y] = true
	b.x++
	if b.x == b.width {
		b.x = 0
		if b.y == b.height-1 {
			b.Scroll()
		} else {
			b.y++
		}
	}
	return nil
}

func (b *Buffer) X() int16 {
	return b.x
}

func (b *Buffer) SetX(x int16) error {
	if x < 0 || x > b.width-1 {
		return errors.New("out of range")
	}
	b.x = x
	return nil
}

func (b *Buffer) Y() int16 {
	return b.y
}

func (b *Buffer) SetY(y int16) error {
	if y < 0 || y > b.height-1 {
		return errors.New("out of range")
	}
	b.y = y
	return nil
}
