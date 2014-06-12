package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"testing"
)

var testURLs = []checkURL{
	{
		Desc:   "test 1",
		Method: "Get",
		URL:    "replace_with_real_url",
		Match: map[string]bool{
			"test1": true,
			"test2": false,
		},
	},
	{
		Desc:   "test 2",
		Method: "Get",
		URL:    "replace_with_real_url",
		Match: map[string]bool{
			"test2": true,
			"test1": false,
		},
	},
}

var testIOOut []byte

type testIO struct{}

func (l testIO) Write(p []byte) (int, error) {
	testIOOut = append(testIOOut, p...)
	return len(p), nil
}

func toDiskTmp(urls []checkURL) (filename string, err error) {
	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return
	}
	fd, err := ioutil.TempFile("", "urler_test_json")
	filename = fd.Name()
	if err != nil {
		return
	}
	fd.Write(data)
	return
}

func TestBody(t *testing.T) {
	expected := []byte("This is the Body")
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, string(expected))
		}))
	defer ts.Close()
	testURLs[0].URL = ts.URL
	body := testURLs[0].body()

	if bytes.Contains(body, expected) {
		t.Logf("body matchs")
	} else {
		t.Fatalf("'%s' != '%s'", expected, body)
	}
}

func TestCheck(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "test1 fake results string")
		}))
	defer ts.Close()
	lfd := testIO{}
	log.SetOutput(lfd)
	testIOOut = nil
	testURLs[0].URL = ts.URL
	testURLs[0].check()

	checkfor := []string{"test 1: pass test1 = true", "test 1: pass test2 = false"}

	for _, re := range checkfor {
		if regexp.MustCompile(re).Match(testIOOut) {
			t.Logf("'%v' found", re)
		} else {
			t.Fatalf("'%v' not found", re)
		}
	}
}

func TestFromDisk(t *testing.T) {
	fname, err := toDiskTmp(testURLs)
	defer os.Remove(fname)
	if err != nil {
		t.Fatalf("Could not write to disk: %s", fname)
	}
	gots := fromDisk(fname)
	if reflect.DeepEqual(testURLs, gots) {
		t.Logf("got matchs")
	} else {
		t.Fatalf("got does not match")
	}

}

func TestMain(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "test1 fake results string")
		}))
	defer ts.Close()
	testURLs[0].URL = ts.URL
	testURLs[1].URL = ts.URL

	fd, err := ioutil.TempFile("", "urler_maintest")
	if err != nil {
		t.Fatal(err)
	}
	*logFile = fd.Name()
	fd.Close()
	defer os.Remove(*logFile)

	*logToStderr = false
	fname, err := toDiskTmp(testURLs)
	defer os.Remove(fname)
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	*urlsFile = fname
	main()

	logFileContents, err := ioutil.ReadFile(*logFile)
	if err != nil {
		t.Fatal(err)
	}
	checkfor := []string{"test 1: pass test1 = true",
		"test 1: pass test2 = false",
		"test 2: fail test2 != true fail",
		"test 2: fail test1 != false fail"}

	for _, re := range checkfor {
		if regexp.MustCompile(re).Match(logFileContents) {
			t.Logf("'%v' found", re)
		} else {
			t.Fatalf("'%v' not found", re)
		}
	}

}
