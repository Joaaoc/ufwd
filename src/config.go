package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFile = "/etc/ufwd/rules.json"

type Rule struct {
	IP    string `json:"ip"`
	Port  string `json:"port"`
	Proto string `json:"proto"`
}

type Config struct {
	Enabled       bool   `json:"enabled"`
	ExternalPorts []Rule `json:"external_ports"`
}

func loadConfig() Config {
	defaultConfig := Config{
		Enabled:       true,
		ExternalPorts: []Rule{},
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return defaultConfig
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return defaultConfig
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return defaultConfig
	}

	return config
}

func saveConfig(config Config) {
	os.MkdirAll(filepath.Dir(configFile), 0755)
	data, _ := json.MarshalIndent(config, "", "    ")
	os.WriteFile(configFile, data, 0644)
}
