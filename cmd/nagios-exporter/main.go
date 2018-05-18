package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func worker(host string) {
	url := fmt.Sprintf("http://%s/nagios/cgi-bin/status.cgi?servicestatustypes=28&hoststatustypes=15", host)
	res, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	body := res.Body
	defer body.Close()

	doc, err := html.Parse(body)
	if err != nil {
		panic(err)
	}

	html.Render(os.Stdout, doc)
}

func main() {
	done := make(chan bool, 1)

	hosts := make(chan string, len(os.Args[1:]))

	for _, host := range os.Args[1:] {
		hosts <- host
	}

	close(hosts)

	go func() {
		for {
			if host, c := <-hosts; c {
				worker(host)
			} else {
				done <- true
			}
		}
	}()

	<-done
}
