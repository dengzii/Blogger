package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
)

var (
	Git  GitConfig
	Blog BlogConfig
)

type GitConfig struct {
	Repo        string
	AccessToken string
	Dir         string
}

type BlogConfig struct {
	Template           string
	Port               int
	Host               string
	WebHookAccessToken string
	Dir                string
}

type config struct {
	Git  GitConfig
	Blog BlogConfig
}

func init() {
	var conf config
	_, err := toml.DecodeFile("config.toml", &conf)
	if err != nil {
		panic(fmt.Sprintf("error on load config: %s", err.Error()))
	}
	Git = conf.Git
	Blog = conf.Blog
}
