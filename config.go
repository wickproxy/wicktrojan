package main

import (
	"github.com/BurntSushi/toml"
)

var config *configPrototype

type configPrototype struct {
	Listen    string
	PanelHost string
	UsageFile string
	Reshape   bool

	Fallback   string
	H2Fallback string

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
		Admin    bool
	}

	Rules []rulePrototype

	Websocket struct {
		Path string
		Host string
	}
}

type rulePrototype struct {
	UDP      bool
	Username string
	Domain   string
	IP       string
	CIDR     string
	Port     string
	Allow    bool
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
	if config.UsageFile != "" {
		usageFile = &config.UsageFile
	}
	initUsers()
	if config.Reshape {
		initShape()
	}
}
