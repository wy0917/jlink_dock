package model

import (
	"os"
	"sync"

	"github.com/pelletier/go-toml/v2"
	log "github.com/wy0917/jlink_dock/logging"
)

type Config struct {
	Server string `toml:"server"` // Server address
	Type   string `toml:"type"`   // Board Type, for example STM32F412Zg
	TTY    string `toml:"tty"`    // Optional: Debug tty port of the board
	ACM    string `toml:"acm"`    // Simulated ACM serial of the board
	Serial string `toml:"serial"` // Serial number of the board
	GDB    struct {
		Server     string `toml:"server"`
		EXEPath    string `toml:"exe_path"`    // /path/to/arm-none-eabi-gdb
		ServerPath string `toml:"server_path"` // /path/to/JLink_Linux_V766b_x86_64
	} `toml:"gdb"`
}

var config *Config
var once sync.Once

func LoadConfig(configFile string) *Config {
	once.Do(func() {
		config = &Config{}
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Info("Config file config.toml not found, reading all values from command line")
		}
		err = toml.Unmarshal(data, &config)
	})

	return config
}
