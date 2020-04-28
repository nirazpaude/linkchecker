package main

import (
	"fmt"
	 "os"
	"flag"
	"encoding/xml"
	 "io"
	 "time"
	 "net/http"
	//  "strings"
)

var inputFile = flag.String("infile", "feed.xml", "Input file path")

type Item struct {
    Link   string `xml:"link"`
}

type result struct {
	index int
	res   http.Response
	err   error
}


func main() {
	start := time.Now()
	flag.Parse()
	xmlFile, err := os.Open(*inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer xmlFile.Close()
	count := 0

	d := xml.NewDecoder(xmlFile)

	semaphoreChan := make(chan struct{}, 1000)
	resultsChan := make(chan *result)

	defer func() {
		close(semaphoreChan)
		close(resultsChan)
	}()
	
	client := &http.Client{}


	for {
		t, tokenErr := d.Token()
		if tokenErr != nil {
			if tokenErr == io.EOF {
			break
			}
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local == "item" {
				var i Item
				if err := d.DecodeElement(&i, &t); err != nil {
				}
				go func(i int, url string) {
					semaphoreChan <- struct{}{}

					req, err := http.NewRequest("GET", url, nil)
					if err != nil {
						fmt.Println(err)
						return
					}
					req.Header.Add("User-Agent", "facebookexternalhit/1.1")
					res, err := client.Do(req)

					if err != nil {
						fmt.Println("getting this error -> ",err)
						fmt.Println(url)
						return
					}

					result := &result{i, *res, err}
					resultsChan <- result
					<-semaphoreChan

					defer res.Body.Close()

				}(count, i.Link)
				count++

			}
		}
	}

	var results []result

	for {
		result := <-resultsChan
		results = append(results, *result)
		fmt.Println(len(results))
		fmt.Println(count)
		if len(results) == count - 1 {
			break
		}
	}


	elapsed := time.Since(start)
    fmt.Println("Async link checking for", count, "links took", elapsed.Seconds(), "seconds")

}