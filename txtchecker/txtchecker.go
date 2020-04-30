package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	// open/prep input file
	start := time.Now()
	inFilename := "../sampleFeed.txt"
	inFile, err := os.Open(inFilename)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	// create/prep results file
	resultsFilename := "../results.csv"
	resultsFile, err := os.Create(resultsFilename)
	if err != nil {
		panic(err)
	}
	defer resultsFile.Close()

	// create channels and worker pool
	// for some reason, leaving off the size argument makes it so the loop stops
	// feeding at the number of workers (e.g. with 8 workers, the feeding loop
	// would only ever push the first 8 lines onto the task channel)
	// TODO: understand why that was happening
	taskChan := make(chan Task, 40000)
	resultChan := make(chan Result, 40000)
	workers := 100
	for w := 0; w < workers; w++ {
		go checkWorker(w+1, taskChan, resultChan)
	}

	// iterate feeder file and push each URL to the channel
	count := 0
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if err = scanner.Err(); err != nil {
			panic(err)
		}
		fmt.Printf("main: feeding record: %+v\n", line)
		taskChan <- Task{count, line}
		count++
	}
	close(taskChan)

	// iterate result channel and write to output CSV
	resultWriter := csv.NewWriter(resultsFile)
	for i := 0; i < count; i++ {
		res := <-resultChan
		err = resultWriter.Write([]string{res.URL, strconv.Itoa(res.Status), strconv.FormatBool(res.Valid), res.ErrorMsg})
		if err != nil {
			fmt.Printf("error writing result %+v: %+v", res, err)
		}
		fmt.Printf("main: result: %+v\n", res)
	}
	close(resultChan)
	resultWriter.Flush()
	if err = resultWriter.Error(); err != nil {
		panic(err)
	}

	elapsed := time.Since(start)
	fmt.Println("Async link checking for", elapsed.Seconds(), "seconds")
}

type Task struct {
	Index int
	URL   string
}

type Result struct {
	Index    int
	URL      string
	Status   int
	Valid    bool
	ErrorMsg string
}

func checkWorker(id int, tasks <-chan Task, r chan<- Result) {
	fmt.Printf("worker %02d: spinning up\n", id)
	// create per-worked http client
	client := &http.Client{}
	for t := range tasks {
		fmt.Printf("worker %02d: received task %+v\n", id, t)
		result := checker(client, t)
		fmt.Printf("worker %02d: completed task with result: %+v\n", id, result)
		r <- result
	}
	fmt.Printf("worker %02d: no more tasks, exiting\n", id)
}

func checker(client *http.Client, t Task) Result {
	result := Result{
		Index:    t.Index,
		URL:      t.URL,
		Status:   -1,
		Valid:    false,
		ErrorMsg: "none",
	}
	req, err := http.NewRequest("GET", t.URL, nil)
	if err != nil {
		result.ErrorMsg = err.Error()
		return result
	}
	req.Header.Add("User-Agent", "facebookexternalhit/1.1")

	res, err := client.Do(req)
	if err != nil {
		result.ErrorMsg = err.Error()
		return result
	}
	defer res.Body.Close()

	result.Status = res.StatusCode
	valid, errmsg := validateResponse(res)
	result.Valid = valid
	result.ErrorMsg = errmsg

	return result
}

func validateResponse(resp *http.Response) (bool, string) {
	if resp.StatusCode != 200 {
		return false, "status was not 200 OK"
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("error reading body: %+v", err)
	}
	if bytes.Contains(bodyBytes, []byte("The page you requested was not found")) {
		return false, "family/product was not found, inactive, or otherwise inaccessible"
	}

	return true, "none"
}
