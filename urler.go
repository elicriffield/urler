package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
)

var urlsFile = flag.String("urlsfile", "urls_list.json",
	"File that must contain a json formated list of urls to check")
var logToStderr = flag.Bool("logtostderr", true,
	"If to log to stderr along with the logfile")
var logFile = flag.String("logfile", "/dev/null", "File to log results to")

type checkURL struct {
	Desc   string          `json:"description"`
	Method string          `json:"method"`
	URL    string          `json:"url"`
	Match  map[string]bool `json:"match"`
}

func (c checkURL) body() ([]byte, error) {
	client := &http.Client{}
	body := []byte{}
	req, err := http.NewRequest(c.Method, c.URL, nil)
	if err != nil {
		return body, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return body, err
	} else {
		body, err = ioutil.ReadAll(resp.Body)
	}
	return body, nil
}

func (c checkURL) check() bool {
	body, err := c.body()
	if err != nil {
		log.Printf("Connection error: %v\n", err)
		return false
	}
	for re, match := range c.Match {
		if got := regexp.MustCompile(re).Match(body); got != match {
			log.Printf("\033[1;31m %s: FAIL %s != %v\033[0m\n", c.Desc, re, match)
		} else {
			log.Printf("\033[1;32m %s: PASS %s = %v\033[0m\n", c.Desc, re, match)
		}
	}
	return true
}

type logIO struct {
	WriteToStderr bool
	LogFile       string
}

func (l logIO) Write(p []byte) (int, error) {
	var fd *os.File
	var err error
	if _, err = os.Stat(l.LogFile); os.IsNotExist(err) {
		fd, err = os.Create(l.LogFile)
	} else if err == nil {
		fd, err = os.OpenFile(l.LogFile, os.O_APPEND|os.O_WRONLY, os.FileMode(0666))
	} else {
		fmt.Fprintf(os.Stderr, "\nOpen log failed: %s\n", err)
		os.Exit(1)
	}

	defer fd.Close()
	writen, err := fd.Write(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nLog write failed: %s\n", err)
		os.Exit(1)
	}
	if l.WriteToStderr {
		fmt.Fprint(os.Stderr, string(p))
	}
	return writen, nil
}

func fromDisk(filename string) (checks []checkURL) {
	fromDisk, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("ReadFile error", err)
	}
	err = json.Unmarshal(fromDisk, &checks)
	if err != nil {
		log.Fatal("json.Unmarshal error ", err)
	}

	return

}

func main() {
	flag.Parse()
	lfd := logIO{
		WriteToStderr: *logToStderr,
		LogFile:       *logFile,
	}
	log.SetOutput(lfd)
	checks := fromDisk(*urlsFile)
	var wg sync.WaitGroup
	for _, check := range checks {
		wg.Add(1)
		go func(ck checkURL) {
			defer wg.Done()
			ck.check()
		}(check)

	}
	wg.Wait()
}
