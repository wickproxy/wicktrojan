package main

import (
	"github.com/BurntSushi/toml"
)

var config *configPrototype

type configPrototype struct {
	Listen    string
	Fallback  string
	PanelHost string

	Rules []string

	TLS struct {
		Certificate string
		PrivateKey  string
		IssueHost   string
		IssueStore  string
		NextProtos  []string
	}

	Users []struct {
		Username string
		Password string
		Quota    int64
	}
}

func loadConfig() {
	config = new(configPrototype)
	if _, err := toml.DecodeFile(*configFile, config); err != nil {
		info("[config] failed to read config from", *configFile, ":", err)
	}
	debug("[config] reading config:", config)
	if len(config.Users) <= 0 {
		fatal("[config] at least one user should exist")
	}
	initUsers()
}
