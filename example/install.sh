#!/bin/bash
set -euo pipefail

function prompt() {
    while true; do
        read -p "$1 [y/N] " yn
        case $yn in
            [Yy] ) return 0;;
            [Nn]|"" ) return 1;;
        esac
    done
}

if [[ $(id -u) != 0 ]]; then
    echo Please run this script as root.
    exit 1
fi

if [[ $(uname -m 2> /dev/null) != x86_64 ]]; then
    echo Please run this script on x86_64 machine.
    exit 1
fi

NAME=wicktrojan
VERSION=$(curl -fsSL https://api.github.com/repos/wickproxy/wicktrojan/releases/latest | grep tag_name | sed -E 's/.*"(.*)".*/\1/' )
FILENAME="wicktrojan-linux-amd64"
DOWNLOADURL="https://github.com/wickproxy/wicktrojan/releases/download/$VERSION/$FILENAME"
TMPDIR="$(mktemp -d)"

INSTALLPREFIX=/usr/local
SYSTEMDPREFIX=/etc/systemd/system

BINARYPATH="$INSTALLPREFIX/bin/$NAME"
CONFIGPATH="$INSTALLPREFIX/etc/$NAME/config.toml"
SYSTEMDPATH="$SYSTEMDPREFIX/$NAME.service"
SYSTEMDPATHTPL="$SYSTEMDPREFIX/$NAME@.service"

echo Entering temp directory $TMPDIR...
cd "$TMPDIR"

echo Downloading $NAME $VERSION...
curl -LO --progress-bar "$DOWNLOADURL" || wget -q --show-progress "$DOWNLOADURL"

echo Installing $NAME $VERSION to $BINARYPATH...
cp "$FILENAME" "$NAME"
install -Dm755 "$NAME" "$BINARYPATH"

echo Installing $NAME server config to $CONFIGPATH...
if ! [[ -f "$CONFIGPATH" ]] || prompt "The server config already exists in $CONFIGPATH, overwrite?"; then
    curl -LO --progress-bar "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/config.toml" || wget -q --show-progress "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/config.toml"
    install -Dm644 config.toml "$CONFIGPATH"
else
    echo Skipping installing $NAME server config...
fi

if [[ -d "$SYSTEMDPREFIX" ]]; then
    echo Installing $NAME systemd service to $SYSTEMDPATH...
    if ! [[ -f "$SYSTEMDPATH" ]] || prompt "The systemd service already exists in $SYSTEMDPATH, overwrite?"; then
        curl -LO --progress-bar "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/wicktrojan.service" || wget -q --show-progress "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/wicktrojan.service"
        install -Dm644 wicktrojan.service "$SYSTEMDPATH"
        echo Reloading systemd daemon...
        systemctl daemon-reload
    else
        echo Skipping installing $NAME systemd service...
    fi

    echo Installing $NAME systemd service template to $SYSTEMDPATHTPL...
    if ! [[ -f "$SYSTEMDPATHTPL" ]] || prompt "The systemd service template already exists in $SYSTEMDPATH, overwrite?"; then
        curl -L -o wicktrojan@.service --progress-bar "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/wicktrojan%40.service" || wget -q --show-progress "https://raw.githubusercontent.com/wickproxy/wicktrojan/$VERSION/example/wicktrojan%40.service"
        install -Dm644 wicktrojan@.service "$SYSTEMDPATHTPL"  || install -Dm644 wicktrojan%40.service "$SYSTEMDPATHTPL"
        echo Reloading systemd daemon...
        systemctl daemon-reload
    else
        echo Skipping installing $NAME systemd service...
    fi
fi

echo Deleting temp directory $TMPDIR...
rm -rf "$TMPDIR"

echo Done!