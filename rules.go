package main

import (
	"net"
	"strings"
)

func initRules() {

}

func match(ctx requestCTX, rule rulePrototype) bool {
	if rule.UDP && !ctx.UDP {
		return false
	}
	if rule.Username != "" && ctx.Username != rule.Username {
		return false
	}

	if rule.Port != "" && ctx.Port != rule.Port {
		return false
	}

	var reqIPs []net.IP
	if ip := net.ParseIP(ctx.Host); ip != nil {
		reqIPs = append(reqIPs, ip)
	} else if ips, err := net.LookupIP(ctx.Host); err == nil {
		reqIPs = ips
	}

	if rule.IP != "" {
		if ruleIP := net.ParseIP(rule.IP); ruleIP != nil {
			flag := false
			for _,ip := range reqIPs {
				if ruleIP.Equal(ip) {
					flag = true
					break
				}
			}
			if !flag {
				return false
			}
		} else {
			info("[rules] warning, can not parse IP", rule.IP)
		}
	}

	if rule.CIDR != "" {
		if _, ruleCIDR, err := net.ParseCIDR(rule.CIDR); err == nil {
			flag := false
			for _,ip := range reqIPs {
				if ruleCIDR.Contains(ip) {
					flag = true
					break
				}
			}
			if !flag {
				return false
			}
		} else {
			info("[rules] warning, can not parse CIDR", rule.CIDR)
		}
	}

	if rule.Domain != "" {
		if rule.Domain == "private" {
			flag := false
			for _, ip := range reqIPs {
				if ip != nil {
					IPv4 := ip.To4()
					if IPv4 != nil && IPv4[0] == 192 && IPv4[1] == 168 {
						flag = true
						break
					} else if IPv4 != nil && (IPv4[0] == 10 || IPv4[0] == 127) {
						flag = true
						break
					} else if IPv4 != nil && IPv4[0] == 172 && (IPv4[1] >= 16 && IPv4[1] <= 31) {
						flag = true
						break
					} else if ip[0] == 0xfd || ip[0] == 0xfe {
						flag = true
						break
					} else if !ip.IsGlobalUnicast() {
						flag = true
						break
					}
				}
			}
			if !flag {
				return false
			}
		} else if !strings.Contains(ctx.Host, rule.Domain) {
			return false
		}
	}
	return true
}

func checkRules(ctx requestCTX) bool {
	if ctx.UDP && ctx.Port == "0" || ctx.Port == "" {
		return true
	}
	for _, rule := range config.Rules {
		if match(ctx, rule) {
			return rule.Allow
		}
	}
	return true
}
