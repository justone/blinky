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
			go arms(p, false, quit, done)
		case command == "arms2":
			go arms(p, true, quit, done)
		case strings.HasSuffix(command, "spin"):
			go spin(p, strings.TrimSuffix(command, "spin"), false, quit, done)
		case strings.HasSuffix(command, "spin2"):
			go spin(p, strings.TrimSuffix(command, "spin2"), true, quit, done)
		default:
			go solid(p, command, quit, done)
		}
		running = true
	}
}

// animate each arm on and then each arm off
func arms(p *piglow.Piglow, reset bool, quit chan bool, done chan bool) {

	var tentacle = 0
	var value = 4

	for {
		select {
		case <-quit:
			done <- true
			return
		default:

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

			time.Sleep(time.Second / 10)
		}
	}
}

// spin through a particular color
func spin(p *piglow.Piglow, color string, reset bool, quit chan bool, done chan bool) {

	leds := colorToLEDs[color]
	var index = 0
	var value = 4

	for {
		select {
		case <-quit:
			done <- true
			return
		default:

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

			time.Sleep(time.Second / 10)
		}
	}
}

// turn on all LEDs of a certain color
func solid(p *piglow.Piglow, color string, quit chan bool, done chan bool) {

	log.Println("setting to ", color)
	switch color {
	case "green":
		p.SetGreen(8)
	case "blue":
		p.SetBlue(8)
	case "white":
		p.SetWhite(8)
	case "yellow":
		p.SetYellow(8)
	case "orange":
		p.SetOrange(8)
	case "red":
		p.SetRed(8)
	case "clear":
	case "all":
		p.SetAll(8)
	default:
		p.SetLED(int8(len(color)%17), 8)
	}
	p.Apply()

	// wait for the end
	for {
		select {
		case <-quit:
			done <- true
			return
		default:
			time.Sleep(time.Second / 10)
		}
	}
}
