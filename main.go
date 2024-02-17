package main

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln("Could not create new wathcer: ", err)
		return
	}
	defer watcher.Close()

	closed := make(chan bool)

	go func() {
		for {
			select {
			case closing := <-closed:
				if closing {
					log.Println("Go func ending")
					break
				}
			default:
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
				default:
				}
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
	closed <- true
	time.Sleep(3 * time.Second)
	log.Println("waited 3 secs")

}

func waitForInput() {
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
