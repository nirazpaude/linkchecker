// use a naive, high-memory approach to deduplicate the original URL feed and
// save it to a saner output (one URL per line)

// depending on the length of the project, an ideal interim status holder would
// be a simple SQLite db and the deliverable would be an export from the DB.
// Using a DB makes it easier to store results as you go and thus be able to not
// have to repeat any previously completed checks when something goes wrong.

package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type XMLItem struct {
	Link string `xml:"link"`
}

func main() {
	// hardcoding filenames due to one-time use nature
	xmlFile := "../feed.xml"
	csvFile := "../feed.txt"

	inFile, err := os.Open(xmlFile)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	outFile, err := os.Create(csvFile)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	// trick for deduping/uniqueness in Go: create a map where the key is the
	// item to be deduped and the value is a bool and use that to store only
	// unique values as the keys
	uniqueLinks := make(map[string]bool)

	d := xml.NewDecoder(inFile)
	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Local != "item" {
				continue
			}
			var i XMLItem
			if err := d.DecodeElement(&i, &t); err != nil {
				panic(err)
			}
			uniqueLinks[i.Link] = true
		}
	}

	// write unique URLs to new feed file
	for url, _ := range uniqueLinks {
		fmt.Println("writing URL", url)
		feedLine := fmt.Sprintf("%s\n", url)
		if _, err := outFile.Write([]byte(feedLine)); err != nil {
			panic(err)
		}
	}
}
