package main

import (
	"github.com/zmap/go-iptree/iptree"
	"os"
	"bufio"
	"strings"
	"net/http"
	"log"
	"sync"
	"time"
	"encoding/json"
	"fmt"
	"io"
	"path"
)

const (
	dumpDownloadTimeout = 30 * time.Second
	dumpUrl = "https://github.com/zapret-info/z-i/raw/master/dump.csv"
	downloadRetryInterval = time.Second * 30
	dumpUpdateInterval = time.Minute * 15
)


func loadDump(path string) (*iptree.IPTree, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	t := iptree.New()
	t.AddByString("0.0.0.0/0", 0)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ";")
		if len(fields) < 2 {
			continue
		}
		for _, ip := range strings.Split(fields[0], "|") {
			t.AddByString(strings.TrimSpace(ip), 1)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return t, err
}


func downloadDump(url, path string) error {
	log.Printf("downloading dump from %s to %s", url, path)
	client := http.Client{Timeout: dumpDownloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		log.Printf("file %s exists, removing it", path)
		if err = os.Remove(path); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf(resp.Status)
	}
	defer resp.Body.Close()
	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	log.Println("dump downloaded")
	return nil
}


func main() {
	listen := os.Args[1]
	dumpDir := os.Args[2]
	lock := sync.Mutex{}

	currentDump := path.Join(dumpDir, "dump.current")
	freshDump := path.Join(dumpDir, "dump.fresh")

	for {
		err := downloadDump(dumpUrl, currentDump)
		if err == nil {
			break
		}
		log.Println(err, "retry after", downloadRetryInterval)
		time.Sleep(downloadRetryInterval)
	}
	db, err := loadDump(currentDump)
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		ticker := time.NewTicker(dumpUpdateInterval).C
		for _ = range ticker {
			log.Println("updating db")
			err := downloadDump(dumpUrl, freshDump)
			if err != nil {
				log.Println(err)
				continue
			}
			newDb, err := loadDump(freshDump)
			if err != nil {
				log.Println(err)
				continue
			}
			lock.Lock()
			db = newDb
			if err = os.Rename(freshDump, currentDump); err != nil {
				log.Println(err)
			}
			log.Println("db updated")
			lock.Unlock()
		}
	}()
	http.HandleFunc("/check_ips", func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		ips := []string{}

		if err := json.NewDecoder(r.Body).Decode(&ips); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(ips) < 1 {
			http.Error(w, "empty ip list", http.StatusBadRequest)
			return
		}
		res := make(map[string]bool, len(ips))

		for _, ip := range ips {
			v, ok, err := db.GetByString(ip)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			res[ip] = false
			if !ok {
				continue
			}
			if flag, _ := v.(int); flag == 1 {
				res[ip] = true
			}
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	log.Println("listening on", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

