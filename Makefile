ifeq ($(OS),Windows_NT)
	builddate = $(shell echo %date:~0,4%%date:~5,2%%date:~8,2%%time:~0,2%%time:~3,2%%time:~6,2%)
else
	builddate = $(shell date +"%Y-%m-%d %H:%M:%S")
endif

Version = 0.1.1
ldflags = -X 'main.version=$(Version)' -X 'main.buildTime=$(builddate)'

build:
	clean
	go build -o build/wicktrojan .

clean:
	rm -f build/wicktrojan*
darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64  go build -ldflags "$(ldflags)" -o build/wicktrojan-darwin-amd64 .
linux-amd64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build -ldflags "$(ldflags)" -o build/wicktrojan-linux-amd64 .
linux-arm64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64  go build -ldflags "$(ldflags)" -o build/wicktrojan-linux-arm64 .
linux-arm:
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm    go build -ldflags "$(ldflags)" -o build/wicktrojan-linux-arm .
linux-mips64:
	CGO_ENABLED=0 GOOS=linux   GOARCH=mips64 go build -ldflags "$(ldflags)" -o build/wicktrojan-linux-mips64 .
windows-x64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build -ldflags "$(ldflags)" -o build/wicktrojan-windows-x64.exe .
windows-x86:
	CGO_ENABLED=0 GOOS=windows GOARCH=386    go build -ldflags "$(ldflags)" -o build/wicktrojan-windows-x86.exe .
freebsd-amd64:
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64  go build -ldflags "$(ldflags)" -o build/wicktrojan-freebsd-amd64 .

all: clean darwin-amd64 linux-amd64 linux-arm64 linux-arm linux-mips64 windows-x64 windows-x86 freebsd-amd64
