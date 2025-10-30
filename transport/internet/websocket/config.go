package websocket

import (
	"math/rand"
	"net/http"
	"time"
	"unicode"

	"github.com/mrst2000/Xray-core/common"
	"github.com/mrst2000/Xray-core/transport/internet" // This is the corrected import path
)


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
	for key, value := range c.Header {
		header.Set(key, value)
	}

	randomizedHost := randomizeCase(c.Host)
	header.Set("Host", randomizedHost)
	header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 11; SM-G981B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.5481.77 Mobile Safari/537.36")
	return header

}

// randomizeCase randomizes the case of letters in a string.
func randomizeCase(s string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	runes := []rune(s)
	for i := range runes {
		if r.Intn(2) == 0 {
			runes[i] = unicode.ToLower(runes[i])
		} else {
			runes[i] = unicode.ToUpper(runes[i])
		}
	}
	return string(runes)
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}