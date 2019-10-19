package horriblefeed

import (
	"github.com/hekmon/transmissionrpc"
	"github.com/spf13/viper"
	"log"
	"regexp"
	"time"
)

const cacheMaxAge = 1 * time.Minute

type transmission struct {
	client        *transmissionrpc.Client
	torrentsCache struct {
		torrents []*transmissionrpc.Torrent
		lastReq  time.Time
	}
}

func newTransmissionClient(config *viper.Viper) (*transmission, error) {
	config.SetDefault("host", "localhost")
	config.SetDefault("port", 9091)

	client, err := transmissionrpc.New(
		config.GetString("host"),
		config.GetString("username"),
		config.GetString("password"),
		&transmissionrpc.AdvancedConfig{
			HTTPS: config.GetBool("ssl"),
			Port:  uint16(config.GetUint("port")),
		},
	)
	if err != nil {
		return nil, err
	}

	return &transmission{
		client: client,
	}, nil
}

func (t *transmission) SeriesMatching(rx *regexp.Regexp) map[string]*transmissionrpc.Torrent {
	series := map[string]*transmissionrpc.Torrent{}

	torrents, err := t.Torrents()
	if err != nil {
		log.Println(err)
		return nil
	}
	for _, torrent := range torrents {
		name := extractName(*torrent.Name, rx)

		if series[name] == nil || series[name].AddedDate.Before(*torrent.AddedDate) {
			series[name] = torrent
		}
	}

	return series
}

func (t *transmission) AddLike(newTorrent string, like *transmissionrpc.Torrent) error {
	_, err := t.client.TorrentAdd(&transmissionrpc.TorrentAddPayload{
		Filename: &newTorrent,

		DownloadDir:       like.DownloadDir,
		BandwidthPriority: like.BandwidthPriority,
	})

	return err
}

func (t *transmission) Torrents() ([]*transmissionrpc.Torrent, error) {
	if t.torrentsCache.torrents != nil && time.Since(t.torrentsCache.lastReq) < cacheMaxAge {
		return t.torrentsCache.torrents, nil
	}

	torrents, err := t.client.TorrentGetAll()
	if err == nil {
		t.torrentsCache.torrents = torrents
	} else {
		t.torrentsCache.torrents = nil
	}

	return torrents, err
}
