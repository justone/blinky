package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	var webqueue string
	if webqueue = os.Getenv("WEBQUEUE"); len(webqueue) == 0 {
		log.Fatal("Set queue with WEBQUEUE env")
	}

	log.Println("Starting queue poll on", webqueue)
	for {
		log.Println("Poll...")
		res, err := http.Get(webqueue)
		if err != nil {
			log.Fatalln("Error:", err)
		}

		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("res:", string(body))
	}
}
