package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"gosuri/uilive"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	configPath = "config.json"
	urlregex   = "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\\+.~#?&//=]*)"
	mp3regex   = "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\\+.~#?&//=]*).mp3"
)

var (
	wg     sync.WaitGroup
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

func download(URL string) (io.ReadCloser, error) {
	res, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}

	return res.Body, nil
}

func save(path string, file io.ReadCloser) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(f, file); err != nil {
		return err
	}

	return nil
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
	writer := uilive.New()
	writer.Start()

	defer writer.Stop()

	res, err := http.Get(params.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != err {
		log.Fatal(err)
	}

	rURL := regexp.MustCompile(urlregex)
	links := rURL.FindAllString(string(body), -1)
	if links != nil {
		return
	}

	sem := make(chan pod)
	wg.Add(params.Goroutines)

	defer close(sem)
	defer wg.Wait()

	fmt.Print("\n # \t STATUS \t SIZE \t NAME")
	for i := 0; i < params.Goroutines; i++ {
		go func() {
			for {
				pod, ok := <-sem
				if !ok {
					wg.Done()
					return
				}

				res, err = http.Get(pod.URL)
				if err != nil {
					log.Fatal(err)
				}
				defer res.Body.Close()

				body, err := ioutil.ReadAll(res.Body)
				if err != err {
					log.Fatal(err)
				}

				rMP3 := regexp.MustCompile(mp3regex)
				songs := rMP3.FindAllString(string(body), -1)
				if songs != nil {
					log.Fatal()
				}

				song := songs[len(songs)-1]
				songNameComponents := strings.Split(song, "/")

				status, fin := "", false
				go func() {
					for i := 1; i <= 3; i++ {
						if !fin {
							status = fmt.Sprintf("downloading%s", strings.Repeat(".", i))
						}

						fmt.Fprintf(writer, "%d/%s \t %s \t %d \t %s \n", pod.index, "n", status, 0, songNameComponents[len(songNameComponents)-1])

						time.Sleep(500 * time.Millisecond)
					}
				}()

				data, err := download(song)
				if err != nil {
					status, fin = "FAILED", true
					log.Print(err)
				}

				if err := save(songNameComponents[len(songNameComponents)-1], data); err != nil {
					status, fin = "FAILED", true
					log.Print(err)
				}

				status, fin = "SUCCESS", true
			}
		}()
	}

	for i, link := range links {
		sem <- pod{i, link}
	}

}
