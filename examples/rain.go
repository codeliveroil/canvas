package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/codeliveroil/canvas"
)

func raindrop(width, height int, c *canvas.Canvas, stop <-chan bool, completed chan<- bool) {
	fps := 15 + rand.Intn(20)
	sleep := func() {
		time.Sleep(time.Duration(1000/fps) * time.Millisecond)
	}

	for {
		select {
		case <-stop:
			completed <- true
			return
		default:
			x := 2 + rand.Intn(width-3)
			rclr := canvas.ColorRandom()
			for y := 0; y < height-1; y++ {
				drawDrop := func(dropClr canvas.Color, style int) {
					c.WriteAtSafe(x, y, dropClr, canvas.ColorDefault, style, "•")
					c.FlushSafe()
				}
				drawDrop(rclr, canvas.StyleNormal)
				sleep()
				if y == height-2 { //to enhance the effect of the drop hitting the ground
					sleep()
				}
				drawDrop(canvas.ColorDefault, canvas.StyleHidden)
			}

			drawSplash := func(x, y int, text string) {
				writeSplash := func(x, y int, clr canvas.Color, style int, text string) {
					c.WriteAtSafe(x, y, clr, canvas.ColorDefault, style, text)
					c.FlushSafe()
				}
				writeSplash(x, y, rclr, canvas.StyleNormal, text)
				sleep()
				writeSplash(x, y, canvas.ColorDefault, canvas.StyleHidden, text)
			}

			drawSplash(x-1, height-3, ". .")
			drawSplash(x-2, height-2, ".   .")
		}
	}
}

func main() {
	width, height := 40, 15

	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	logger := log.New(f, "", log.LstdFlags)
	logger.Println("Starting...")

	// Set up canvas
	c := canvas.NewCanvas(width, height, canvas.ColorDefault)
	c.Logger = logger
	canvas.HideCursor()
	defer canvas.ShowCursor()
	canvas.DisableEcho()
	defer canvas.EnableEcho()
	c.Move(10, 14)
	c.SetForeground(canvas.ColorWhite)
	c.Write("Press Ctrl+C to quit")
	c.Flush()

	// Start drawing
	stop := make(chan bool)      //indicate to threads to stop
	completed := make(chan bool) //threads can indicate that they are done after receiving the 'stop' signal
	drops := 4

	for i := 0; i < drops; i++ {
		go raindrop(width, height, c, stop, completed)
	}

	// Trap Ctrl+C and wait for drawing threads to complete current iteration
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt)
	go func() {
		<-ctrlc
		for i := 0; i < drops; i++ {
			stop <- true
		}
	}()
	for i := 0; i < drops; i++ {
		<-completed
	}

	//Clean up
	c.SetBackground(canvas.ColorDefault)
	c.Clear()
	c.Move(0, 0) // move cursor to 0,0 to make the canvas vanish seamlessly and display the prompt
	c.Flush()    //FlushSafe() is not required because both drawing threads have completed

}
