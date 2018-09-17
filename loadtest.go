package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

var reportInterval int
var baseURL string
var filters []string
var keepCookies bool

func main() {
	fileName := flag.String("l", "", "file name of access log file to interpret")
	goroutines := flag.Int("c", 2, "number of concurrent requests")
	filtersArg := flag.String("f", "", "comma-separated list of URLs to filter")
	repIntArg := flag.Int("r", 1, "report interval in seconds")
	basUrlArg := flag.String("b", "", "base URL as prefix in front of paths in access logs")
	keepCookiesArg := flag.Bool("k", false, "boolean value keep cookies or not")

	flag.Parse()

	filters = strings.Split(*filtersArg, ",")
	reportInterval = *repIntArg
	baseURL = *basUrlArg
	keepCookies = *keepCookiesArg

	log.Println("Access log file:", *fileName)
	log.Println("Concurrent req #:", *goroutines)
	log.Println("Filters:", filters)
	log.Printf("Report every: %d seconds", reportInterval)
	log.Println("Base URL", baseURL)
	log.Println("Keep cookies", keepCookies)

	queue := make(chan string)
	timer := make(chan int)
	hits := make(chan int)
	stop := make(chan struct{})

	for i := 0; i < *goroutines; i++ {
		go readFromQueue(i, queue, stop, hits)
	}

	go report(hits, timer)
	go sleepSec(reportInterval, timer, stop)

	readLogs(*fileName, queue)

	close(stop)

	os.Exit(0)
}

func sleepSec(timeout int, timer chan int, stop chan struct{}) {
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

func readFromQueue(id int, queue chan string, stop chan struct{}, hits chan int) {

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	for {
		processLogLine(<-queue, hits, client)
	}
}

func processLogLine(logLine string, hits chan int, client *http.Client) {

	start := strings.Index(logLine, "\"GET ")
	end := strings.Index(logLine, " HTTP/")

	if start > 0 && end > 0 {
		path := logLine[start+5 : end]

		filtered := false

		for _, el := range filters {
			if strings.Index(path, el) > 0 {
				filtered = true
			}
		}

		if !filtered {

			r, err := client.Get(baseURL + path)

			if err != nil {
				log.Println(err)
			} else {
				hits <- 1
				respURL, err := url.Parse(baseURL + path)
				if err != nil {
					log.Println("Could not read cookies")
					log.Fatal(err)
				} else if keepCookies {
					client.Jar.SetCookies(respURL, r.Cookies())
				}
			}
		}
	}
}

func readLogs(fileName string, queue chan string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		queue <- scanner.Text()
	}
}
