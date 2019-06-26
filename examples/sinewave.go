package main

import (
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/codeliveroil/canvas"
)

func drawSineWave(width, cycles, direction, fps int, c *canvas.Canvas, waveClr canvas.Color, stop <-chan bool, completed chan<- bool) {
	const amplitude = 5
	waveY := make([]int, width, width)
	for frame := 0.5; frame < math.MaxFloat64; frame += 0.5 {
		select {
		case <-stop:
			completed <- true
			return
		default:
			// Create sine way
			for x := 0; x < width; x++ {
				y := int(amplitude * math.Sin(float64(x*2*cycles)*(math.Pi/float64(width))+(frame*float64(direction))))
				waveY[x] = y + amplitude //move the 0-line to the center of the canvas
			}

			// Wave drawing helper
			render := func(clr canvas.Color) {
				for x := 0; x < width; x++ {
					c.SetForegroundSafe(clr)
					c.SetSafe(x, waveY[x], '•')
				}
				c.FlushSafe()
			}

			// Render wave
			render(waveClr)
			time.Sleep(time.Duration(1000/fps) * time.Millisecond)
			//time.Sleep(1 * time.Second)
			// Erase wave before rendering next to create animation effect
			render(c.Background)
		}
	}
}

func main() {
	width, height := 60, 13

	// Set up canvas
	c := canvas.NewCanvas(width, height, canvas.ColorBlack)
	canvas.HideCursor()
	defer canvas.ShowCursor()
	canvas.DisableEcho()
	defer canvas.EnableEcho()

	c.Move(20, 12)
	c.SetForeground(canvas.ColorWhite)
	c.Write("Press Ctrl+C to quit")

	// Start drawing
	stop := make(chan bool)      //indicate to threads to stop
	completed := make(chan bool) //threads can indicate that they are done after receiving the 'stop' signal

	go drawSineWave(width, 1, +1, 20, c, canvas.ColorYellow, stop, completed)
	go drawSineWave(width, 3, -1, 15, c, canvas.ColorRed, stop, completed)

	// Trap Ctrl+C and wait for drawing threads to complete current iteration
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)
	go func() {
		<-ctrlc
		for i := 0; i < 2; i++ {
			stop <- true
		}
	}()
	for i := 0; i < 2; i++ {
		<-completed
	}

	//Clean up
	c.SetBackground(canvas.ColorDefault)
	c.Clear()
	c.Move(0, 0) // move cursor to 0,0 to make the canvas vanish seamlessly and display the prompt
	c.Flush()    //FlushSafe() is not required because both drawing threads have completed
}
