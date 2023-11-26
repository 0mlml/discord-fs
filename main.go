package main

import (
	"flag"
	"log"

	"github.com/0mlml/cfgparser"
)

var (
	config     *cfgparser.Config
	configPath = flag.String("config", "discord-fs.cfg", "Path to config file")
)

func main() {
	flag.Parse()

	defaultConfig := &cfgparser.Config{}
	defaultConfig.Literal(
		map[string]bool{},
		map[string]string{
			"discord_token": "YOUR_TOKEN # Generate a token here: https://discord.com/developers/applications",
			"server_id":     "111111111111111111 # The server to generate files in",
			"your_key":      "YOUR_KEY # The key to encrypt files with",
		},
		map[string]int{
			"max_file_size": 24214400,
		},
		map[string]float64{},
	)

	cfgparser.SetDefaultConfig(defaultConfig)

	config = &cfgparser.Config{}
	if err := config.From(*configPath); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	if !setToken(config.String("discord_token")) {
		log.Fatalf("Error setting token")
	}

	intialize()

	log.Printf("Ready\n")

	readPump()
}
