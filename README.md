# Wicktrojan

Wicktrojan is a trojan server that is written in Golang and is under `MIT License`.

## Features
* UDP/TCP mode
* with or without transport layer security
* automatic certificates issue
* One script to install on linux amd64 machiens
* rule check based on IP, CIDR, domain, port, username and protocol
* quota or usage limitation
## Install

via curl
```
sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/wickproxy/wicktrojan/main/example/install.sh)"
```
via wget
```
sudo bash -c "$(wget -O- https://raw.githubusercontent.com/wickproxy/wicktroja/main/example/install.sh)"
```

Or download binary manually from: [Release Page](https://github.com/wickproxy/wicktrojan/releases)

## Usage

Command line usage:
```
wicktrojan -help # print help
wicktrojan -version # print version
wicktrojan -config <config.toml> [-usage usage.db] [-log logging.txt]
```

Please refer to [`example/config.toml`](https://github.com/wickproxy/wicktrojan/blob/main/example/config.toml) to see how to configure.

## Build
Prerequisites:
* `Golang` 1.12 or above
* `git` to clone this repository

It is easy to build Wicktrojan using `go` command:
```
git clone https://github.com/wickproxy/wicktrojan
go build -o build/wicktrojan .
```

Another way to compile Wicktrojan is to use `Make` command:
```
make <platform>       # to build for special platform. Including: linux-amd64, linux-arm64 , darwin-amd64, windows-x64, windows-x86 and freebsd-amd64
make all        # to build for all three platforms
```