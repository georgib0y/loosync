package main

import (
	"bufio"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("Could not create new wathcer: ", err)
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Fatalln("Error reading event")
					return
				}

				log.Println(event)
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Fatalln("Error reading error")
					return
				}

				log.Println(err)
			}

		}
	}()

	log.Println("Adding folder")

	err = watcher.Add("/tmp")
	if err != nil {
		log.Fatalln("Error adding new dir to watcher: ", err)
		return
	}

	log.Println("wating for input")
	waitForInput()
	log.Println("got input")

	watcher.Close()
	log.Println("Closed")
}

func waitForInput() {
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
