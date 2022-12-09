package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	configPath = "config.json"
	urlregex   = `http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`
	mp3regex   = `http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+.mp3`
)

var (
	params configParams
)

type configParams struct {
	Goroutines int    `json:"goroutines"`
	Downloads  string `json:"downloads-directory"`
	URL        string `json:"url"`
}

type pod struct {
	index int
	URL   string
}

// func download(URL string) (io.ReadCloser, error) {
// 	res, err := http.Get(URL)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode != http.StatusOK {
// 		return nil, errors.New(res.Status)
// 	}

// 	return res.Body, nil
// }

func save(path string, URL string) (int, error) {
	res, err := http.Get(URL)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, errors.New(res.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if _, err = io.Copy(f, res.Body); err != nil {
		return 0, err
	}

	sdata, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	return len(sdata), nil
}

func init() {
	_, err := ioutil.ReadFile(configPath)

	// Check if config.json exists, otherwise create it
	if err != nil {
		fmt.Println(configPath, " doesn't exist. Creating file...")

		data := configParams{
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
	res, err := http.Get(params.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != err {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`(?m)http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`)
	links := re.FindAllString(string(body), -1)

	fmt.Println("Found ", len(links), " links on the front page.")

	fmt.Print("\n # \t STATUS \t SIZE \t NAME \n")
	for i, link := range links {
		res, err = http.Get(link)
		if err != nil {
			log.Fatal("1", err)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != err {
			log.Fatal("2", err)
		}

		rMP3 := regexp.MustCompile(`http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+.mp3`)
		songs := rMP3.FindAllString(string(body), -1)
		if songs == nil {
			continue
		}

		song := songs[len(songs)-1]
		songNameComponents := strings.Split(song, "/")
		out, err := os.Create(fmt.Sprint(params.Downloads, "/", songNameComponents[len(songNameComponents)-1]))

		res, err = http.Get(song)
		if err != nil {
			fmt.Printf("%d/%d \t %s \t %d \t %s \n", i, len(links), "FAILED", 0, songNameComponents[len(songNameComponents)-1])
			continue
		}

		n, err := io.Copy(out, res.Body)
		if err != nil {
			fmt.Printf("%d/%d \t %s \t %d \t %s \n", i, len(links), "FAILED", 0, songNameComponents[len(songNameComponents)-1])
			continue
		}

		fmt.Printf("%d/%d \t %s \t %d \t %s \n", i, len(links), "SUCCESS", n, songNameComponents[len(songNameComponents)-1])

		out.Close()
		res.Body.Close()
	}

}
