// Copyright (c) 2018 codeliveroil. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package canvas

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	ErrorOutOfBounds = errors.New("out of bounds")
	ErrorTruncated   = errors.New("string truncated")
)

// Canvas is a representation of a painting canvas on a terminal that
// supports 256 colors. The canvas is rendered in-line.
//
// It provides functions to manipulate the canvas and thread-safe
// implementations of the same functions as well. If you don't intend
// to use this in a multi-threaded fashion, then use the non
// thread-safe functions as they offer better performance which is
// critical in animation and gaming applications.
type Canvas struct {
	// Width is the width of the canvas
	Width int

	// Height is the height of the canvas
	Height int

	// Logger can be used for debugging purposes.
	Logger *log.Logger

	Background Color

	// CursorOnEnd will move the cursor to (x-Max, y-Max) when Flush()
	// is invoked. This comes in handy if you want the user prompt to
	// show up right after the canvas (as opposed to somewhere in
	// between), when there is an abrupt termination.
	CursorOnEnd bool

	buf   bytes.Buffer
	mutex sync.Mutex

	x     int
	y     int
	fg    Color
	bg    Color
	style int
}

// Color defines a color either expressed as an RGBA value or as one of
// the 256 colors on a terminal that supports 256 colors.
type Color struct {
	// RGBA defines a standard color value defined as R,G,B,A. The
	// system will pick the closest color on a terminal that supports
	// 256 colors.
	RGBA color.Color

	// Term256 can be used to specify one of the 256 terminal colors
	// on a terminal that supports 256 colors. This value will be used
	// only if RGBA is nil.
	Term256 uint8

	isDefault bool
}

var (
	ColorDefault      = Color{isDefault: true}
	ColorBlack        = Color{Term256: 0}
	ColorRed          = Color{Term256: 1}
	ColorGreen        = Color{Term256: 2}
	ColorYellow       = Color{Term256: 3}
	ColorBlue         = Color{Term256: 4}
	ColorMagenta      = Color{Term256: 5}
	ColorCyan         = Color{Term256: 6}
	ColorLightGray    = Color{Term256: 7}
	ColorDarkGray     = Color{Term256: 8}
	ColorLightRed     = Color{Term256: 9}
	ColorLightGreen   = Color{Term256: 10}
	ColorLightYellow  = Color{Term256: 11}
	ColorLightBlue    = Color{Term256: 12}
	ColorLightMagenta = Color{Term256: 13}
	ColorLightCyan    = Color{Term256: 14}
	ColorWhite        = Color{Term256: 15}
)

func ColorRandom() Color {
	return Color{Term256: uint8(rand.Intn(256))}
}

const (
	StyleNoChange = 1 << iota
	StyleNormal
	StyleBold
	StyleDim
	StyleUnderlined
	StyleBlink
	StyleInverted
	StyleHidden
)

const escape = "\033"

// NewCanvas returns a new Canvas object
func NewCanvas(width, height int, background Color) *Canvas {
	c := &Canvas{
		Width:      width,
		Height:     height,
		Background: background,
	}
	c.SetBackground(background)
	c.SetStyle(StyleNormal)
	for i := 0; i < c.Height; i++ {
		for i := 0; i < c.Width; i++ {
			c.buf.WriteString(" ")
		}
		if i < c.Height-1 {
			c.SetBackground(ColorDefault) //to ensure that [width,terminalWidth) is not colored in background color
			c.buf.WriteString("\n")
			c.SetBackground(background)
		}
	}
	c.x, c.y = width, height-1 //this is the only time c.x is out of bounds, legally.
	c.Move(0, 0)
	c.Flush()
	return c
}

func (c *Canvas) Clear() {
	c.Move(0, 0)
	for i := 0; i < c.Height; i++ {
		for i := 0; i < c.Width; i++ {
			c.Write(" ")
		}
		if i < c.Height-1 {
			c.Move(0, i+1)
		}
	}
	c.Move(0, 0)
}

func (c *Canvas) ClearSafe() {
	c.mutex.Lock()
	c.Clear()
	c.mutex.Unlock()
}

func (c *Canvas) SetBackground(clr Color) {
	if c.bg == clr {
		return
	}
	c.setBgFg(clr, true)
	c.bg = clr
}

func (c *Canvas) SetBackgroundSafe(clr Color) {
	c.mutex.Lock()
	c.SetBackground(clr)
	c.mutex.Unlock()

}

func (c *Canvas) SetForeground(clr Color) {
	if c.fg == clr {
		return
	}
	c.setBgFg(clr, false)
	c.fg = clr
}

func (c *Canvas) SetForegroundSafe(clr Color) {
	c.mutex.Lock()
	c.SetForeground(clr)
	c.mutex.Unlock()
}

func (c *Canvas) setBgFg(clr Color, bg bool) {
	var op string
	if bg {
		op = "4"
	} else {
		op = "3"
	}
	if clr.isDefault {
		c.buf.WriteString(escape + "[")
		c.buf.WriteString(op)
		c.buf.WriteString("9m")
		return
	}
	var x256Clr int
	if clr.RGBA != nil {
		x256Clr = Colors.Index(clr.RGBA)
	} else {
		x256Clr = int(clr.Term256)
	}
	c.buf.WriteString(escape + "[")
	c.buf.WriteString(op)
	c.buf.WriteString("8;5;")
	c.buf.WriteString(strconv.Itoa(x256Clr))
	c.buf.WriteString("m")
}

func (c *Canvas) SetStyle(style int) {

	if c.style == style {
		return
	}
	if style&StyleNormal != 0 {
		c.buf.WriteString(escape + "[0m")
	}
	if style&StyleBold != 0 {
		c.buf.WriteString(escape + "[1m")
	}
	if style&StyleDim != 0 {
		c.buf.WriteString(escape + "[2m")
	}
	if style&StyleUnderlined != 0 {
		c.buf.WriteString(escape + "[4m")
	}
	if style&StyleBlink != 0 {
		c.buf.WriteString(escape + "[5m")
	}
	if style&StyleInverted != 0 {
		c.buf.WriteString(escape + "[7m")
	}
	if style&StyleHidden != 0 {
		c.buf.WriteString(escape + "[8m")
	}
	c.style = style
}

func (c *Canvas) SetStyleSafe(style int) {
	c.mutex.Lock()
	c.SetStyle(style)
	c.mutex.Unlock()
}

func (c *Canvas) Set(x, y int, char rune) {
	if err := c.Move(x, y); err == nil {
		c.Write(string(char))
	}
}

func (c *Canvas) SetSafe(x, y int, char rune) {
	c.mutex.Lock()
	c.Set(x, y, char)
	c.mutex.Unlock()
}

func (c *Canvas) Move(x, y int) error {
	if c.x == x && c.y == y {
		return nil
	}

	if x > c.Width-1 || x < 0 || y > c.Height-1 || y < 0 {
		c.errorF("out of bounds: (%d,%d)", x, y)
		return ErrorOutOfBounds
	}
	move := func(curr, given int, forwardOp string, backwardOp string) {
		write := func(num int, op string) {
			c.buf.WriteString(escape + "[" + strconv.Itoa(num) + op)
		}
		if curr < given {
			write(given-curr, forwardOp)
		} else if curr > given {
			write(curr-given, backwardOp)
		}
	}

	move(c.x, x, "C", "D")
	move(c.y, y, "B", "A")
	c.x, c.y = x, y
	return nil
}

func (c *Canvas) MoveSafe(x, y int) error {
	c.mutex.Lock()
	err := c.Move(x, y)
	c.mutex.Unlock()
	return err
}

func (c *Canvas) Write(text string) {
	if r, l := c.Width-c.x, utf8.RuneCountInString(text); l > r {
		orig := text
		text = text[0:r] //TODO: fix this so that runes are truncated not in the middle of their bytes
		c.errorF("string truncated: %s", orig)
	}
	c.buf.WriteString(text)
	c.x += utf8.RuneCountInString(text)
	if c.x >= c.Width {
		c.Move(c.Width-1, c.y)
	}
}

func (c *Canvas) WriteSafe(text string) {
	c.mutex.Lock()
	c.Write(text)
	c.mutex.Unlock()
}

func (c *Canvas) WriteAt(x, y int, foreground, background Color, style int, text string) {
	if err := c.Move(x, y); err != nil {
		return
	}
	c.SetStyle(style)
	c.SetForeground(foreground)
	c.SetBackground(background)
	c.Write(text)
}

func (c *Canvas) WriteAtSafe(x, y int, foreground, background Color, style int, text string) {
	c.mutex.Lock()
	c.WriteAt(x, y, foreground, background, style, text)
	c.mutex.Unlock()
}

func (c *Canvas) Flush() {
	if c.CursorOnEnd {
		c.Move(c.Width-1, c.Height-1)
	}
	fmt.Printf(c.buf.String())
	c.buf.Reset()
}

func (c *Canvas) FlushSafe() {
	c.mutex.Lock()
	c.Flush()
	c.mutex.Unlock()
}

func (c *Canvas) errorF(msg string, args ...interface{}) {
	if c.Logger != nil {
		c.Logger.Printf("[error] "+msg, args...)
	}
}

func (c *Canvas) debugF(msg string, args ...interface{}) {
	if c.Logger != nil {
		c.Logger.Printf("[debug] "+msg, args...)
	}
}

func HideCursor() {
	fmt.Printf(escape + "[?25l")
}

func ShowCursor() {
	fmt.Printf(escape + "[?25h")
}

func DisableEcho() {
	toggleEcho("-")
}

func EnableEcho() {
	toggleEcho("")
}

func toggleEcho(prefix string) {
	cmd := exec.Command("stty", prefix+"echo")
	cmd.Stdin = os.Stdin
	cmd.Run()
}

func main() {
	// Set up logger
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)
	logger.Println("Starting...")

	// Set up canvas
	c := NewCanvas(3, 4, ColorGreen)
	c.Logger = logger
	c.CursorOnEnd = true
	HideCursor()
	DisableEcho()
	c.SetBackground(Color{Term256: 238})
	c.SetForeground(Color{Term256: 125})
	c.Set(1, 1, 'H')
	c.Write("ello")
	c.Write("X")
	c.Flush()
	time.Sleep(3 * time.Second)
	c.Move(1, 10)
	time.Sleep(100 * time.Millisecond)
	c.Clear()
	c.Flush()
	//time.Sleep(50000 * time.Millisecond)
	c.debugF("Animating")
	for i := 0; i < 256; i++ {
		c.SetBackground(Color{Term256: uint8(i)})
		c.Clear()
		c.Flush()
		time.Sleep(1 * time.Millisecond)
	}

	ShowCursor()
	EnableEcho()
	time.Sleep(3 * time.Second)
}
