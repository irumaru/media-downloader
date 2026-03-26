package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Database DatabaseConfig  `yaml:"database"`
	Ytdlp    YtdlpConfig     `yaml:"ytdlp"`
	Channels []ChannelConfig `yaml:"channels"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	StaticDir string `yaml:"static_dir"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type YtdlpConfig struct {
	Path        string `yaml:"path"`
	AudioFormat string `yaml:"audio_format"`
}

type ChannelConfig struct {
	Secret    string `yaml:"secret"`
	Name      string `yaml:"name"`
	OutputDir string `yaml:"output_dir"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
