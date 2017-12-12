package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xAX/notificator"
	"github.com/fsnotify/fsnotify"
)

func debounce(interval time.Duration, input chan string) chan string {
	output := make(chan string)
	go func() {
		lastItem := ""
		for {
			select {
			case item := <-input:
				if lastItem != item && lastItem != "" {
					output <- item
				}
				lastItem = item
			case <-time.After(interval):
				if lastItem != "" {
					output <- lastItem
					lastItem = ""
				}
			}
		}
	}()
	return output
}

func checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	spammyFiles := make(chan string)
	files := debounce(time.Second, spammyFiles)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if !strings.HasSuffix(event.Name, ".crdownload") {
					spammyFiles <- event.Name
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	go func() {
		notify := notificator.New(notificator.Options{
			DefaultIcon: "icon/default.png",
			AppName:     "My test App",
		})

		for {
			path := <-files
			if sha256, err := checksum(path); err != nil {
				log.Println(path, "error:", err)
			} else {
				log.Println(path, "sha256:", sha256)
				notify.Push("Downloaded checksums", fmt.Sprintf("%s\nsha256: %s", path, sha256), "/home/user/icon.png", notificator.UR_NORMAL)
			}
		}
	}()

	err = watcher.Add(filepath.Join(os.Getenv("HOME"), "Downloads"))
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
