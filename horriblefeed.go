package main

import (
	"flag"
	"fmt"
	"github.com/hekmon/transmissionrpc"
	"github.com/mmcdole/gofeed"
	"log"
	"os"
	"regexp"
	"time"
)

func main() {
	host := flag.String("host", "localhost", "Transmission URL")
	user := flag.String("user", "transmission", "Transmission user")
	password := flag.String("pass", "transmission", "Transmission password")
	horriblefeed := flag.String("horriblefeed", "https://www.horriblesubs.info/rss.php?res=1080", "HorribleSubs feed url")
	maxage := flag.Duration("maxage", 2*7*24*time.Hour, "Stop reading horrible feed when this age is reached")
	daemonize := flag.Bool("daemonize", false, "Run as a daemon in a loop")

	flag.Parse()

	transmission, err := transmissionrpc.New(*host, *user, *password, &transmissionrpc.AdvancedConfig{
		HTTPS: true,
		Port:  443,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	parser := gofeed.NewParser()

	run := true
	for run {
		torrents, err := transmission.TorrentGetAll()
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		series := map[string]*transmissionrpc.Torrent{}

		for _, t := range torrents {
			sName := seriesName(*t.Name)
			if sName != "" {
				addTorrent(series, sName, t)
			}
		}

		feed, err := parser.ParseURL(*horriblefeed)
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}

		for _, item := range feed.Items {
			if time.Since(*item.PublishedParsed) > *maxage {
				break
			}

			previousTorrent := series[seriesName(item.Title)]
			if previousTorrent == nil || previousTorrent.AddedDate.After(*item.PublishedParsed) {
				continue
			}

			_, err := transmission.TorrentAdd(&transmissionrpc.TorrentAddPayload{
				DownloadDir: previousTorrent.DownloadDir,
				Filename:    &item.Link,
			})

			if err != nil {
				log.Printf("Error adding %s: %s\n", item.Title, err.Error())
				continue
			}

			log.Printf("Added %s\n", item.Title)
		}

		run = *daemonize
		if run {
			log.Println("End of feed reached, sleeping")
			time.Sleep(10 * time.Minute)
		}
	}
}

var seriesNameRegex = regexp.MustCompile(`\[[Hh]orrible[Ss]ubs\] ?(.+) ?- \d+ \[\d+p?\](?:\.\w{1,5})?`)

func seriesName(chapterName string) string {
	matches := seriesNameRegex.FindStringSubmatch(chapterName)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}

func addTorrent(torrents map[string]*transmissionrpc.Torrent, key string, value *transmissionrpc.Torrent) {
	if torrents[key] == nil || torrents[key].AddedDate.Before(*value.AddedDate) {
		torrents[key] = value
	}
}
