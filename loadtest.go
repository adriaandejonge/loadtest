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

type Options struct {
	reportInterval *int
	baseURL        *string
	filters        *string
	keepCookies    *bool
	goroutines     *int
	verbose        *bool
	suppressErr    *bool
	repeat         *bool
	fileName       *string
}

func (o *Options) Filters() []string {
	return strings.Split(*o.filters, ",")
}

var options = Options{
	flag.Int("r", 1, "report interval in seconds"),
	flag.String("b", "", "base URL as prefix in front of paths in access logs"),
	flag.String("f", "a,b,c", "comma-separated list of URLs to filter"),
	flag.Bool("k", false, "keep cookies across requests"),
	flag.Int("c", 2, "number of concurrent requests"),
	flag.Bool("v", false, "verbose output"),
	flag.Bool("s", false, "suppress errors"),
	flag.Bool("rp", false, "repeat after done reading log file"),
	flag.String("l", "", "file name of access log file to interpret"),
}

func main() {

	flag.Parse()

	log.Println("Access log file:", *options.fileName)
	log.Println("Concurrent req #:", *options.goroutines)
	log.Println("Filters:", options.Filters())
	log.Printf("Report every: %d seconds", *options.reportInterval)
	log.Println("Base URL", *options.baseURL)
	log.Println("Keep cookies", *options.keepCookies)
	log.Println("Verbose output", *options.verbose)
	log.Println("Suppress errors", *options.suppressErr)
	log.Println("Repeat after done with log file", *options.repeat)

	queue := make(chan string)
	timer := make(chan int)
	hits := make(chan int)
	stop := make(chan struct{})

	for i := 0; i < *options.goroutines; i++ {
		go readFromQueue(i, queue, stop, hits)
	}

	go report(hits, timer)
	go sleepSec(timer, stop)

	if *options.repeat {
		for {
			readLogs(*options.fileName, queue)
		}
	} else {
		readLogs(*options.fileName, queue)
	}

	close(stop)

	os.Exit(0)
}

func sleepSec(timer chan int, stop chan struct{}) {
	for {
		time.Sleep(time.Duration(*options.reportInterval) * time.Second)
		timer <- *options.reportInterval

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

	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100

	if *options.goroutines > 100 {
		defaultTransport.MaxIdleConns = *options.goroutines
		defaultTransport.MaxIdleConnsPerHost = *options.goroutines
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

		for _, el := range options.Filters() {
			if strings.Index(path, el) >= 0 {
				filtered = true
			}
		}

		if !filtered {

			fullPath := *options.baseURL + path

			if *options.verbose {
				log.Println("URL", fullPath)
			}

			r, err := client.Get(fullPath)
			if err != nil {
				if !*options.suppressErr {
					log.Println(err)
				}
			} else {
				hits <- 1
				io.Copy(ioutil.Discard, r.Body)
				r.Body.Close()
				respURL, err := url.Parse(fullPath)
				if err != nil {
					log.Println("Could not read cookies")
					log.Fatal(err)
				} else if *options.keepCookies {
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
