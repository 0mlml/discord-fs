package main

import (
	"flag"

	"github.com/0mlml/cfgparser"
)

var (
	config     *cfgparser.Config
	configPath = flag.String("config", "discord-fs.cfg", "Path to config file")
	logger     = NewLogger()
)

func main() {
	flag.Parse()

	defaultConfig := &cfgparser.Config{}
	defaultConfig.Literal(
		map[string]bool{
			"advanced_terminal": true,
		},
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
		logger.Printf("Error parsing config file: %v", err)
		return
	}

	if !setToken(config.String("discord_token")) {
		logger.Printf("Error setting token")
		return
	}

	intialize()

	logger.Printf("Ready\n")

	if config.Bool("advanced_terminal") {
		advancedReadPump()
	} else {
		simpleReadPump()
	}
}
