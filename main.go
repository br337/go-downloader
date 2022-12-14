package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gosuri/uilive"
)

const (
	configPath = "config.json"
	urlregex   = `http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`
	mp3regex   = `http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+.mp3`
)

var (
	params  ConfigParams
	mp3s    map[string]bool = make(map[string]bool)
	visited map[string]bool = make(map[string]bool)
)

type ConfigParams struct {
	Goroutines int    `json:"goroutines"`
	Downloads  string `json:"downloads-directory"`
	URL        string `json:"url"`
}

func CollectLinks(link string, depth int) {
	if depth == 0 {
		return
	}

	writer := uilive.New()
	writer.Start()

	visited[link] = true

	fmt.Fprintf(writer, "[%s]  %d \t %s \n", "LOADING", depth, link)

	res, err := http.Get(link)
	if err != nil {
		fmt.Fprintf(writer, "[%s]  %d \t %s \n", "FAILED", depth, link)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != err {
		fmt.Fprintf(writer, "[%s]  %d \t %s \n", "FAILED", depth, link)
	}

	data := string(body)

	re := regexp.MustCompile(urlregex)
	rMP3 := regexp.MustCompile(mp3regex)
	links := re.FindAllString(data, -1)
	songs := rMP3.FindAllString(data, -1)

	res.Body.Close()

	fmt.Fprintf(writer, "[%s]  %d \t %s \n", "SUCCESS", depth, link)
	writer.Flush()
	writer.Stop()

	for _, song := range songs {
		mp3s[song] = true
	}

	for _, v := range links {
		if visited[v] {
			continue
		}

		visited[v] = true
		CollectLinks(v, depth-1)
	}
}

func download(URL string) {
	writer := uilive.New()
	writer.Start()
	writer.Flush()

	songNameComponents := strings.Split(URL, "/")
	out, err := os.Create(fmt.Sprint(params.Downloads, "/", songNameComponents[len(songNameComponents)-1]))

	fmt.Fprintf(writer, "[%s]  ?? \t %s \n", "LOADING", songNameComponents[len(songNameComponents)-1])
	writer.Flush()

	res, err := http.Get(URL)
	if err != nil {
		fmt.Fprintf(writer, "[%s]  %d \t %s \n", "FAILED", 0, songNameComponents[len(songNameComponents)-1])
		writer.Flush()
	}

	n, err := io.Copy(out, res.Body)
	if err != nil {
		fmt.Fprintf(writer, "[%s]  %d \t %s \n", "FAILED", 0, songNameComponents[len(songNameComponents)-1])
		writer.Flush()
	}
	fmt.Fprintf(writer, "[%s]  %.2fmb \t %s \n", "SUCCESS", float64(n)/(1<<20), songNameComponents[len(songNameComponents)-1])
	writer.Flush()

	out.Close()
	res.Body.Close()
	writer.Flush()
	writer.Stop()
}

func init() {
	_, err := ioutil.ReadFile(configPath)

	// Check if config.json exists, otherwise create it
	if err != nil {
		fmt.Println(configPath, " doesn't exist. Creating file...")

		data := ConfigParams{
			URL:        "http://blog.livedoor.jp/daibakarenji/infoinfo.html",
			Downloads:  "Downloads",
			Goroutines: 5,
		}

		defaultParams, err := json.MarshalIndent(data, "", " ")
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(configPath, defaultParams, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	data, err := ioutil.ReadFile(configPath)
	if err := json.Unmarshal(data, &params); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully loaded ", configPath)

	err = os.MkdirAll(params.Downloads, os.ModePerm)
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("Created directory ", params.Downloads)
		} else {
			log.Fatal(err)
		}
	}

	fmt.Println("Loaded directory ", params.Downloads)

}

func main() {

	guard := make(chan struct{}, params.Goroutines)
	signal := make(chan struct{})

	fmt.Printf("\n   STATUS  D     NAME \n")
	CollectLinks(params.URL, 2)

	fmt.Printf("\nFound %d mp3 files \n", len(mp3s))

	fmt.Printf("\n   STATUS  SIZE \t NAME \n")
	for k := range mp3s {
		guard <- struct{}{}
		go func(k string) {
			download(k)
			<-guard
		}(k)

		time.Sleep(time.Second * 2)
	}

	<-signal
}
