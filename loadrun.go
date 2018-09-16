package github.com/adriaandejonge/loadrun

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	BASEURL = "https://PUTYOURURLHERE"
)

var filterURLs = []string{
	"/filter-this",
	"/and-this",
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Need at least 2 command line args")
	}
	fileName := os.Args[1]
	goroutines, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	log.Println("LOG:", fileName)
	log.Println("# Goroutines:", goroutines)

	queue := make(chan string)
	timer := make(chan int)
	hits := make(chan int)

	for i := 0; i < goroutines; i++ {
		go readFromQueue(i, queue)
	}

	go report(hits, timer)
	go sleepSec(5, timer)

	readLogs(fileName, queue, hits)

	for i := 0; i < goroutines; i++ {
		queue <- "END"
	}

	os.Exit(0)

}

func sleepSec(timeout int, timer chan int) {
	for {
		time.Sleep(time.Duration(timeout) * time.Second)
		timer <- timeout
	}
}

func report(hits chan int, timer chan int) {
	counter := 0
	previousCount := 0
	for {
		select {
		case count := <-hits:
			counter += count
		case seconds := <-timer:
			new := counter - previousCount
			new /= seconds
			previousCount = counter
			log.Println(strconv.Itoa(new) + " req/s")
		}
	}
}

func readFromQueue(id int, queue chan string) {
	logLine := <-queue

	for logLine != "END" {
		start := strings.Index(logLine, "\"GET ")
		end := strings.Index(logLine, " HTTP/")

		url := "not found"

		if start > 0 && end > 0 {
			url = logLine[start+5 : end]

			filtered := false

			for _, el := range filterURLs {
				if strings.Index(url, el) > 0 {
					filtered = true
				}
			}

			if !filtered {

				_, err := http.Get(BASEURL + url)
				if err != nil {
					log.Println(err)
				}

			}
		}

		logLine = <-queue
	}

}

func readLogs(fileName string, queue chan string, hits chan int) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		queue <- scanner.Text()
		hits <- 1
	}
}
