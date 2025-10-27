package websocket

import (
	"math/rand"
	"net/http"
	"time"
	"unicode"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/transport/internet"
)

const protocolName = "websocket"

// A list of modern mobile user agents to be chosen from randomly.
var mobileUserAgents = []string{
	"Mozilla/5.0 (Linux; Android 13; SM-S908B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Android 13; Mobile; rv:109.0) Gecko/114.0 Firefox/114.0",
	"Mozilla/5.0 (Linux; Android 12; SM-A205U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Mobile Safari/537.36 SamsungBrowser/19.0",
}

// getRandomMobileUserAgent selects a random user agent from the mobileUserAgents list.
func getRandomMobileUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	return mobileUserAgents[rand.Intn(len(mobileUserAgents))]
}

// GetNormalizedPath returns a normalized path of this websocket config.
func (c *Config) GetNormalizedPath() string {
	path := c.Path
	if len(path) == 0 {
		return "/"
	}
	if path[0] != '/' {
		return "/" + path
	}
	return path
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	// Apply user-defined headers first.
	for _, h := range c.Header {
		header.Set(h.Key, h.Value)
	}

	// Randomize the case of the Host header.
	randomizedHost := randomizeCase(c.Host)
	header.Set("Host", randomizedHost)

	// Set a random mobile User-Agent, overwriting if one was already set.
	header.Set("User-Agent", getRandomMobileUserAgent())

	return header
}

// randomizeCase randomizes the case of letters in a string.
func randomizeCase(s string) string {
	rand.Seed(time.Now().UnixNano())
	runes := []rune(s)
	for i, r := range runes {
		if rand.Intn(2) == 0 {
			runes[i] = unicode.ToLower(r)
		} else {
			runes[i] = unicode.ToUpper(r)
		}
	}
	return string(runes)
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}