package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
var goroutines int
var verbose bool
var suppressErr bool
var repeat bool

func main() {
	fileName := flag.String("l", "", "file name of access log file to interpret")
	goroutinesArg := flag.Int("c", 2, "number of concurrent requests")
	filtersArg := flag.String("f", "", "comma-separated list of URLs to filter")
	repIntArg := flag.Int("r", 1, "report interval in seconds")
	basUrlArg := flag.String("b", "", "base URL as prefix in front of paths in access logs")
	keepCookiesArg := flag.Bool("k", false, "boolean value keep cookies or not")
	verboseArg := flag.Bool("v", false, "boolean for verbose output")
	suppressArg := flag.Bool("s", false, "boolean to suppress errors")
	repeatArg := flag.Bool("rp", false, "repeat after done reading log file")

	flag.Parse()

	filters = strings.Split(*filtersArg, ",")
	reportInterval = *repIntArg
	baseURL = *basUrlArg
	keepCookies = *keepCookiesArg
	goroutines = *goroutinesArg
	verbose = *verboseArg
	suppressErr = *suppressArg
	repeat = *repeatArg

	log.Println("Access log file:", *fileName)
	log.Println("Concurrent req #:", goroutines)
	log.Println("Filters:", filters)
	log.Printf("Report every: %d seconds", reportInterval)
	log.Println("Base URL", baseURL)
	log.Println("Keep cookies", keepCookies)
	log.Println("Verbose output", verbose)
	log.Println("Suppress errors", suppressErr)
	log.Println("Repeat after done with log file", repeat)

	queue := make(chan string)
	timer := make(chan int)
	hits := make(chan int)
	stop := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go readFromQueue(i, queue, stop, hits)
	}

	go report(hits, timer)
	go sleepSec(reportInterval, timer, stop)

	if repeat {
		for {
			readLogs(*fileName, queue)
		}
	} else {
		readLogs(*fileName, queue)
	}

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

	// Customize the Transport to have larger connection pool
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100

	if goroutines > 100 {
		defaultTransport.MaxIdleConns = goroutines
		defaultTransport.MaxIdleConnsPerHost = goroutines
	}

	client := &http.Client{
		Jar:       jar,
		Transport: &defaultTransport,
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

			if verbose {
				log.Println("URL", baseURL+path)
			}

			r, err := client.Get(baseURL + path)
			if err != nil {
				if !suppressErr {
					log.Println(err)
				}
			} else {
				hits <- 1
				io.Copy(ioutil.Discard, r.Body)
				r.Body.Close()
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
