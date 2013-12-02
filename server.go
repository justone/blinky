package main

import (
	"github.com/wjessop/go-piglow"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	var webqueue string
	if webqueue = os.Getenv("WEBQUEUE"); len(webqueue) == 0 {
		log.Fatal("Set queue with WEBQUEUE env")
	}

	var commandChan = make(chan string)

	go func(in chan string) {
		var p *piglow.Piglow
		var err error

		// Create a new Piglow
		p, err = piglow.NewPiglow()
		if err != nil {
			log.Fatal("Couldn't create a Piglow: ", err)
		}

		p.SetAll(0)
		err = p.Apply()
		if err != nil { // Apply the changes
			log.Fatal("Couldn't apply changes: ", err)
		}

		for {
			command := <-in

			log.Println("processing command:", command)

			p.SetAll(0)
			switch command {
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
				p.SetLED(int8(len(command)%17), 8)
			}
			err = p.Apply()
			if err != nil { // Apply the changes
				log.Fatal("Couldn't apply changes: ", err)
			}
		}
	}(commandChan)

	log.Println("Starting queue poll on", webqueue)
	for {
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

			commandChan <- strings.TrimSpace(string(body))
		}
	}
}
