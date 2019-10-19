package horriblefeed

import (
	"github.com/mmcdole/gofeed"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
)

type HorribleFeed struct {
	transmission *transmission

	feedParser *gofeed.Parser
	feedsLock  sync.Mutex
	feeds      []feed
	maxFeedAge time.Duration

	defaultRegex *regexp.Regexp
	log          *log.Logger
}

type feed struct {
	url   string
	regex *regexp.Regexp
}

const defaultRegex = `\[[\w\d-_ ]+\] ?(.+) - \d+`

func New(config *viper.Viper) (*HorribleFeed, error) {
	transmission, err := newTransmissionClient(config.Sub("transmission"))
	if err != nil {
		return nil, errors.WithMessage(err, "cannot connect to transmission")
	}

	hf := &HorribleFeed{
		transmission: transmission,
		feedParser:   gofeed.NewParser(),
		defaultRegex: regexp.MustCompile(defaultRegex),
		log:          log.New(os.Stderr, "", 0),
		maxFeedAge:   2 * 7 * 24 * time.Hour, // 2 weeks old
	}
	err = hf.UseFeeds(config)
	if err != nil {
		return nil, err
	}

	return hf, nil
}

func (hf *HorribleFeed) UseFeeds(config *viper.Viper) error {
	var feedsConfig struct {
		Feeds []struct {
			URL   string
			Regex string
		}
	}

	err := config.Unmarshal(&feedsConfig)
	if err != nil {
		return errors.WithMessage(err, "could not load feeds config")
	}

	newFeeds := make([]feed, 0, len(feedsConfig.Feeds))
	for _, f := range feedsConfig.Feeds {
		var rx *regexp.Regexp
		if f.Regex != "" {
			rx, err = regexp.Compile(f.Regex)
			if err != nil {
				return errors.WithMessagef(err, "could not load feeds config, regex `%s` does not compile", f.Regex)
			}
		} else {
			rx = hf.defaultRegex
		}

		newFeeds = append(newFeeds, feed{
			url:   f.URL,
			regex: rx,
		})
	}

	hf.feedsLock.Lock()
	hf.feeds = newFeeds
	hf.feedsLock.Unlock()

	return nil
}

func (hf *HorribleFeed) ParseAndAdd() {
	hf.feedsLock.Lock()
	defer hf.feedsLock.Unlock()

	for _, feed := range hf.feeds {
		parsedFeed, err := hf.feedParser.ParseURL(feed.url)
		if err != nil {
			hf.log.Println(errors.WithMessagef(err, "error parsing '%s'", feed.url))
			continue
		}

		hf.log.Printf("Parsing %s...\n", parsedFeed.Title)

		series := hf.transmission.SeriesMatching(feed.regex)
		for _, item := range parsedFeed.Items {
			if time.Since(*item.PublishedParsed) > hf.maxFeedAge {
				// Assume feed is ordered chronologically, so one item older than threshold implies following items are older as well
				break
			}

			prevTorrent := series[extractName(item.Title, feed.regex)]

			// Discard untracked and already added torrents
			if prevTorrent == nil || prevTorrent.AddedDate.After(*item.PublishedParsed) {
				continue
			}

			err := hf.transmission.AddLike(item.Link, prevTorrent)
			if err != nil {
				hf.log.Println(errors.WithMessagef(err, "error adding %s:", item.Title))
			} else {
				hf.log.Printf("Added %s\n", item.Title)
			}
		}
	}
}

// extractName extracts the series name from a chapter and a regex. Regex is asumed to capture the name in $1
func extractName(chapter string, rx *regexp.Regexp) string {
	matches := rx.FindStringSubmatch(chapter)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
