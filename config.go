package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BotApiUrl    string `yaml:"bot_api_url"`
	BotToken     string `yaml:"bot_token"`
	ServerDomain string `yaml:"server_domain"`
	Port         int    `yaml:"port"`
	Secret       string `yaml:"secret"`
	OwnerId      int64  `yaml:"owner_id"`
	ChatId       int64  `yaml:"chat_id"`
	LogThreadId  int64  `yaml:"log_thread_id"`
}

func parseConfig() (*Config, error) {
	file, err := os.Open("config.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
