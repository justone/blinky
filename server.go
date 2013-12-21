package main

import (
	"github.com/wjessop/go-piglow"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// from go-piglow
var colorToLEDs = map[string][3]int8{
	"white":  [3]int8{12, 9, 10},
	"blue":   [3]int8{14, 4, 11},
	"green":  [3]int8{3, 5, 13},
	"yellow": [3]int8{2, 8, 15},
	"orange": [3]int8{1, 7, 16},
	"red":    [3]int8{0, 6, 17},
}

type Blinky struct {
	p    *piglow.Piglow
	quit chan bool
	done chan bool
}

func main() {
	var webqueue string
	if webqueue = os.Getenv("WEBQUEUE"); len(webqueue) == 0 {
		log.Fatal("Set queue with WEBQUEUE env")
	}

	var commandChan = make(chan string)

	// start up command dispatcher
	go dispatcher(commandChan)

	log.Println("Starting queue poll on", webqueue)
	for {
		log.Println("Poll...")
		res, err := http.Get(webqueue)
		if err != nil {
			log.Fatalln("Error:", err)
		}

		if res.StatusCode == 200 {
			body, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Fatal(err)
			}

			// send command over to dispatcher
			commandChan <- strings.TrimSpace(string(body))
		}
	}
}

func dispatcher(in chan string) {
	var p *piglow.Piglow
	var err error

	var quit = make(chan bool)
	var done = make(chan bool)
	var running = false

	// Create a new Piglow
	p, err = piglow.NewPiglow()
	if err != nil {
		log.Fatal("Couldn't create a Piglow: ", err)
	}

	var blinky = &Blinky{p, quit, done}

	// clear the LEDs
	p.SetAll(0)
	err = p.Apply()
	if err != nil { // Apply the changes
		log.Fatal("Couldn't apply changes: ", err)
	}

	for {
		command := <-in

		log.Println("processing command:", command)

		// if animation is already running, stop it
		// and wait for it to finish
		if running {
			quit <- true
			<-done
		}

		// clear all LEDs
		p.SetAll(0)
		err = p.Apply()

		// dispatch command to sub-goroutines
		switch {
		case command == "arms":
			go arms(blinky, false)
		case command == "arms2":
			go arms(blinky, true)
		case strings.HasSuffix(command, "spin"):
			go spin(blinky, strings.TrimSuffix(command, "spin"), false)
		case strings.HasSuffix(command, "spin2"):
			go spin(blinky, strings.TrimSuffix(command, "spin2"), true)
		default:
			go solid(blinky, command)
		}
		running = true
	}
}

// animate each arm on and then each arm off
func arms(blinky *Blinky, reset bool) {

	var tentacle = 0
	var value = 4

	animate(blinky, time.Second/10, func(p *piglow.Piglow) {
		if tentacle == 3 {
			tentacle = 0

			if !reset {
				if value == 4 {
					value = 0
				} else {
					value = 4
				}
			}
		}

		if reset {
			p.SetAll(0)
			p.Apply()
		}
		p.SetTentacle(tentacle, uint8(value))
		p.Apply()

		// next tentacle
		tentacle += 1
	})
}

// spin through a particular color
func spin(blinky *Blinky, color string, reset bool) {

	leds := colorToLEDs[color]
	var index = 0
	var value = 4

	animate(blinky, time.Second/10, func(p *piglow.Piglow) {
		if index == 3 {
			index = 0

			if !reset {
				if value == 4 {
					value = 0
				} else {
					value = 4
				}
			}
		}

		if reset {
			p.SetAll(0)
			p.Apply()
		}
		p.SetLED(int8(leds[index]), uint8(value))
		p.Apply()

		// next index
		index += 1
	})
}

// This function handles the common case of a simple animation with no cleanup
func animate(blinky *Blinky, timeout time.Duration, callback func(*piglow.Piglow)) {
	for {
		select {
		case <-blinky.quit:
			blinky.done <- true
			return
		default:
			callback(blinky.p)
			time.Sleep(timeout)
		}
	}
}

// turn on all LEDs of a certain color
func solid(blinky *Blinky, color string) {

	log.Println("setting to ", color)
	switch color {
	case "green":
		blinky.p.SetGreen(8)
	case "blue":
		blinky.p.SetBlue(8)
	case "white":
		blinky.p.SetWhite(8)
	case "yellow":
		blinky.p.SetYellow(8)
	case "orange":
		blinky.p.SetOrange(8)
	case "red":
		blinky.p.SetRed(8)
	case "clear":
	case "all":
		blinky.p.SetAll(8)
	default:
		blinky.p.SetLED(int8(len(color)%17), 8)
	}
	blinky.p.Apply()

	// wait for the end, aka no animation
	animate(blinky, time.Second/10, func(p *piglow.Piglow) {})
}
