package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mb         int64 = 1024 * 1024
	gb         int64 = 1024 * 1024 * 1024
	flushTimer       = time.Second * 300
)

var (
	usageLock sync.RWMutex
)

type userStruct struct {
	Hex      string `json:"-"`
	Username string `json:"username"`
	Password string `json:"-"`
	Quota    int64  `json:"-"`
	Usage    int64  `json:"usage"`
	Admin    bool   `json:"-"`
}

var users map[string]userStruct

func parseHex(password string) string {
	h224 := sha256.Sum224([]byte(password))
	return hex.EncodeToString(h224[:])
}

func initUsers() {

	tmpStore := make(map[string]int64)
	if *usageFile != "" {
		fp, err := os.Open(*usageFile)
		if err == nil {
			bufReader := bufio.NewReader(fp)
			for {
				strs, _, err := bufReader.ReadLine()
				if err != nil {
					break
				}
				raw := strings.Split(string(strs), " ")
				if len(raw) == 2 {
					if usageI, err := strconv.ParseInt(raw[1], 10, 64); err == nil {
						tmpStore[raw[0]] = usageI
					}
				}
			}
		}
		fp.Close()
	}

	users = make(map[string]userStruct)
	for _, u := range config.Users {
		if u.Username == "" || u.Password == "" {
			fatal("[config] user must have username and password")
		}
		idx := parseHex(u.Password)
		var usage int64 = 0
		if _, ok := tmpStore[idx]; ok {
			usage = tmpStore[idx]
		}
		users[idx] = userStruct{
			Username: u.Username,
			Password: u.Password,
			Hex:      parseHex(u.Password),
			Quota:    u.Quota * gb,
			Usage:    usage,
			Admin:    u.Admin,
		}
	}
	if *usageFile != "" {
		go storeUsage()
	}
}

func authenticate(reqhex string) (username string, ok bool) {
	// avoid undefined behavior
	ok = false
	username = ""
	if u, exists := users[reqhex]; exists {
		username = u.Username
		usageLock.RLock()
		ok = u.Quota == 0 || u.Usage < u.Quota
		usageLock.RUnlock()
	}
	return username, ok
}

func updateUsage(ctx requestCTX) {
	usageLock.Lock()
	if _, ok := users[ctx.Hex]; ok {
		u2 := users[ctx.Hex]
		u2.Usage += (ctx.RUsage + ctx.SUsage)
		users[ctx.Hex] = u2
	}
	usageLock.Unlock()
}

func checkUsage(hex string) bool {
	if u, ok := users[hex]; ok {
		return u.Usage < u.Quota
	}
	return true
}

func storeUsage() {

	d := time.Duration(flushTimer)
	t := time.NewTicker(d)

	for {
		<-t.C
		debug("[quota] store usage into database")
		fp, err := os.Create(*usageFile)
		if err != nil {
			fatal("[quote] can not create usage file")
		}
		bufWrtier := bufio.NewWriter(fp)
		usageLock.RLock()
		for _, u := range users {
			bufWrtier.WriteString(fmt.Sprintf("%v %v\n", u.Hex, u.Usage))
		}
		bufWrtier.WriteString("\n")
		usageLock.RUnlock()
		bufWrtier.Flush()
		fp.Close()
	}
}
