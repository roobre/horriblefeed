package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"roob.re/horriblefeed"
	"time"
)

func main() {
	config := viper.New()
	config.SetConfigName("horriblefeed")
	config.AddConfigPath(".")
	config.AddConfigPath("$XDG_CONFIG_HOME/horriblefeed")
	config.AddConfigPath("$HOME/.config/horriblefeed")

	config.SetDefault("intervalMinutes", 10)

	if err := config.ReadInConfig(); err != nil {
		log.Fatal(errors.WithMessage(err, "could not read config file"))
	}

	hf, err := horriblefeed.New(config)
	if err != nil {
		log.Fatal(err)
	}

	config.WatchConfig()
	config.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Reloading config file...")
		if hf.UseFeeds(config) == nil {
			log.Println("Feeds reloaded successfully.")
		} else {
			log.Println("Error reloading feeds. No changes were made.")
		}
	})

	for {
		hf.ParseAndAdd()
		time.Sleep(time.Duration(config.GetInt64("intervalMinutes")) * time.Minute)
	}
}
