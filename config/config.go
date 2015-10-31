package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"github.com/darvik80/monitor/log"
)

const (
	configFile = "settings.json"
)

type Config struct {
	DaemonMode  bool   `json:"daemon"`
	LogFile string `json:"log-file"`
	WatchDir    string `json:"watch-dir"`
	TorrentsDir string `json:"torrents-dir"`
}

func NewConfig() *Config {
	return &Config{
		false,
		"",
		"/tmp",
		"/tmp/torrents",
	}
}

func (this *Config) Read(path []string) error {
	var (
		err  error
		file []byte
	)
	for _, folder := range path {
		filePath := filepath.Join(folder, configFile)
		file, err = ioutil.ReadFile(filePath)
		if err == nil {
			log.Debugf("Use settings from %s\n", filePath)
			break
		}
	}

	if err == nil {
		err = json.Unmarshal(file, this)
	}
	return err
}
