package config

import (
	"os"

	"github.com/naoina/toml"
)

type Config struct {
	Port string
	KeystorePath string
	KeystorePassword string
	GpingList []Gping
}

type Gping struct {
	Url string // json rpc url
	Address string
	VaultAddress string
}

func NewConfig(file string) *Config {
	c := new(Config)

	if file, err := os.Open(file); err != nil {
		panic(err)
	} else {
		defer file.Close()
		if err := toml.NewDecoder(file).Decode(c); err != nil {
			panic(err)
		}
		return c 
	}
	return nil 
}
