package main

import (
	"github.com/wjessop/go-piglow"

	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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

var colorOrder = [6]string{
	"white",
	"blue",
	"green",
	"yellow",
	"orange",
	"red",
}

var animations = map[string]func(*Blinky){
	"shimmer: turn random LEDs to random brightnesses":       func(b *Blinky) { shimmer(b) },
	"pulse: pulse all LEDs up and down":                      func(b *Blinky) { pulse(b) },
	"bounce: bounce a single LED up and down all arms":       func(b *Blinky) { bounce(b, false) },
	"bounce2: bounce a single LED each arm in turn":          func(b *Blinky) { bounce(b, true) },
	"cycle: turn all LEDs on and then off in bands":          func(b *Blinky) { cycle(b) },
	"arms: light each arm in turn and then turn of each arm": func(b *Blinky) { arms(b, false) },
	"arms2: light each arm in turn by itself":                func(b *Blinky) { arms(b, true) },
}

var animationsColor = map[string]func(*Blinky, string){
	"<color>spin: spin through the LEDs of the specified color":  func(b *Blinky, color string) { spin(b, color, false) },
	"<color>spin2: spin through the LEDs of the specified color": func(b *Blinky, color string) { spin(b, color, true) },
	"<color>: turn the specified color LED on":                   func(b *Blinky, color string) { solid(b, color) },
}

type Blinky struct {
	p    *piglow.Piglow
	quit chan bool
	done chan bool
}

func main() {
	var commandChan = make(chan string)

	// start up command dispatcher
	go dispatcher(commandChan)

	var webqueue string
	if webqueue = os.Getenv("WEBQUEUE"); len(webqueue) > 0 {
		log.Println("WEBQUEUE env variable found, polling", webqueue)
		for {
			res, err := http.Get(webqueue)
			if err != nil {
				log.Fatalln("Error:", err)
			}

			body, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Fatal(err)
			}

			if res.StatusCode == 200 {
				// send command over to dispatcher
				commandChan <- strings.TrimSpace(string(body))
			}
		}
	} else {
		var animation = flag.String("a", "cycle", "specify an animation to run (default: cycle)")
		var list = flag.Bool("l", false, "list available animations")
		flag.Parse()

		if *list {
			fmt.Println("\nAvailable animations:")
			for desc, _ := range animations {
				fmt.Println("  ", desc)
			}
			for desc, _ := range animationsColor {
				fmt.Println("  ", desc)
			}

			fmt.Println("\nAvailable colors:")
			for _, color := range colorOrder {
				fmt.Println("  ", color)
			}

			fmt.Println("")
		} else {
			commandChan <- *animation

			var sleepForever = make(chan int)
			<-sleepForever
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

		log.Println("Starting animation:", command)

		// if animation is already running, stop it
		// and wait for it to finish
		if running {
			quit <- true
			<-done
		}

		// clear all LEDs
		p.SetAll(0)
		err = p.Apply()

		running = false

		if !running {
		ANIM:
			for key, value := range animations {
				if strings.HasPrefix(key, command+":") {
					running = true
					go value(blinky)
					break ANIM
				}
			}
		}

		if !running {
		ANIMCOLOR:
			for key, value := range animationsColor {
				if strings.Contains(key, "<color>") {
					for _, color := range colorOrder {
						if strings.HasPrefix(key, strings.Replace(command, color, "<color>", -1)+":") {
							running = true
							go value(blinky, color)
							break ANIMCOLOR
						}
					}
				}
			}
		}

		if !running {
			log.Println("Can't understand animation", command)
		}
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

// cycle leds from the center
func cycle(blinky *Blinky) {

	var index = 0
	var value = 4

	animate(blinky, time.Second/10, func(p *piglow.Piglow) {
		if index == len(colorOrder) {
			index = 0

			if value == 4 {
				value = 0
			} else {
				value = 4
			}
		}

		for _, led := range colorToLEDs[colorOrder[index]] {
			p.SetLED(led, uint8(value))
		}
		p.Apply()

		// next index
		index += 1
	})
}

// pulse all LEDs
func pulse(blinky *Blinky) {

	var step = 2
	var max = 30
	var value = 2
	var brighten = true

	animate(blinky, time.Second/10, func(p *piglow.Piglow) {

		if value == max {
			brighten = false
		}
		if value == 2 {
			brighten = true
		}

		p.SetAll(uint8(value))
		p.Apply()

		if brighten {
			value += step
		} else {
			value -= step
		}
	})
}

// shimmer all LEDs
func shimmer(blinky *Blinky) {

	var min = 2
	var max = 10

	var init = func(p *piglow.Piglow) {
		p.SetAll(uint8(min))
		p.Apply()
	}

	var animate = func(p *piglow.Piglow) {
		p.SetLED(int8(rand.Intn(18)), uint8(rand.Intn(max-min)+min))
		p.Apply()
	}

	animateWithInit(blinky, time.Second/50, init, animate)
}

// bounce a single led along the arm(s)
func bounce(blinky *Blinky, singleArm bool) {

	var index = 0
	var value = 4
	var arm = 0
	var outward = true

	animate(blinky, time.Second/10, func(p *piglow.Piglow) {

		if index == (len(colorOrder) - 1) {
			outward = false
		}
		if index == 0 {
			outward = true

			// advance arm if in single arm mode
			if singleArm {
				arm += 1
				if arm == 3 {
					arm = 0
				}
			}
		}

		p.SetAll(0)
		p.Apply()
		for i, led := range colorToLEDs[colorOrder[index]] {
			if !singleArm || arm == i {
				p.SetLED(led, uint8(value))
			}
		}
		p.Apply()

		// next index
		if outward {
			index += 1
		} else {
			index -= 1
		}
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

// animateWithInit is for animations that need initialization
func animateWithInit(blinky *Blinky, timeout time.Duration, init func(*piglow.Piglow), animation func(*piglow.Piglow)) {

	init(blinky.p)

	animate(blinky, timeout, animation)
}

// turn on all LEDs of a certain color
func solid(blinky *Blinky, color string) {

	var init = func(p *piglow.Piglow) {
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
	}

	// wait for the end, aka no animation
	animateWithInit(blinky, time.Second/10, init, func(p *piglow.Piglow) {})
}
