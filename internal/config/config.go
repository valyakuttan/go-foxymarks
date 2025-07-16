package config

import (
	"encoding/json"
	"log"
	"os"
)

type ConfigData struct {
	Sources  map[string]Source
	RepoPath string
}

type config struct {
	DataSources []Source `json:"sources"`
	RepoPath    string   `json:"repo"`
}

type Source struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s Source) String() string {
	return s.Name
}

func WriteToConfig(cfgFile string, cfg ConfigData) {
	var sources []Source
	for _, s := range cfg.Sources {
		sources = append(sources, s)
	}
	c := config{DataSources: sources, RepoPath: cfg.RepoPath}

	out, err := os.Create(cfgFile)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	if _, err = out.Write(data); err != nil {
		panic(err)
	}
	if err = out.Sync(); err != nil {
		panic(err)
	}
}

func ReadFromConfig(cfgFile string) ConfigData {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		log.Fatalf("Reading from config file %q failed: %v", cfgFile, err)
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}
	m := make(map[string]Source)
	for _, s := range cfg.DataSources {
		// expand shell varialbes
		s.Path = os.ExpandEnv(s.Path)
		m[s.Name] = s
	}
	cfg.RepoPath = os.ExpandEnv(cfg.RepoPath)

	return ConfigData{Sources: m, RepoPath: cfg.RepoPath}
}
