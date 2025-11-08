// THIS IS THE CORRECTLY MODIFIED FILE.
// PLEASE REPLACE THE ORIGINAL WITH THIS.

package websocket

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"io"
	"math/big"
	gonet "net"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/mrst2000/Xray-core/common"
	"github.com/mrst2000/Xray-core/common/errors"
	"github.com/mrst2000/Xray-core/common/net"
	"github.com/mrst2000/Xray-core/transport/internet"
	"github.com/mrst2000/Xray-core/transport/internet/browser_dialer"
	"github.com/mrst2000/Xray-core/transport/internet/stat"
	"github.com/mrst2000/Xray-core/transport/internet/tls"
)

// ========= START: ADDED HELPER FUNCTIONS =========

// randomizeCase randomizes the case of each letter in a string.
func randomizeCase(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	for _, r := range s {
		n, err := rand.Int(rand.Reader, big.NewInt(2))
		// Fallback to original case if random source fails
		if err != nil {
			builder.WriteRune(r)
			continue
		}
		if n.Int64() == 0 {
			builder.WriteRune(unicode.ToLower(r))
		} else {
			builder.WriteRune(unicode.ToUpper(r))
		}
	}
	return builder.String()
}

// caseSensitiveHeaderKeys contains a list of header keys whose values are
// known to be case-sensitive. We will not randomize the values for these headers.
// The keys are in uppercase for case-insensitive lookup.
var caseSensitiveHeaderKeys = map[string]bool{
	"AUTHORIZATION":            true,
	"PROXY-AUTHORIZATION":      true,
	"COOKIE":                   true,
	"SET-COOKIE":               true,
	"SEC-WEBSOCKET-KEY":        true,
	"SEC-WEBSOCKET-ACCEPT":     true,
	"SEC-WEBSOCKET-PROTOCOL":   true,
	"SEC-WEBSOCKET-EXTENSIONS": true,
}

// applyHeaderModifications modifies the request header map to have a fixed User-Agent and randomized header cases.
// It returns a new http.Header object because header keys are canonicalized, preventing in-place key case changes.
func applyHeaderModifications(originalHeader http.Header) http.Header {
	// 1. Set the static User-Agent in the original map first to ensure it's included.
	const ua = "Mozilla/5.0 (Linux; Android 11; SM-G981B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.5481.77 Mobile Safari/537.36"
	originalHeader.Set("User-Agent", ua)

	// The "Host" header is special and handled by gorilla/websocket from the URL.
	// We will handle its case randomization separately if needed, but gorilla handles it.
	// For other headers, we create a new map to allow for non-canonical keys.

	newHeader := make(http.Header)
	for key, values := range originalHeader {
		randomizedKey := randomizeCase(key)

		// Check if the value is case sensitive based on the original, canonical key
		isSensitive := caseSensitiveHeaderKeys[strings.ToUpper(key)]

		if isSensitive {
			// If value is sensitive, don't change its case, only the key's case.
			newHeader[randomizedKey] = values
		} else {
			// If value is not sensitive, randomize its case along with the key's case.
			randomizedValues := make([]string, len(values))
			for i, v := range values {
				randomizedValues[i] = randomizeCase(v)
			}
			newHeader[randomizedKey] = randomizedValues
		}
	}

	return newHeader
}

// ========= END: ADDED HELPER FUNCTIONS =========

// Dial dials a WebSocket connection to the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (stat.Connection, error) {
	errors.LogInfo(ctx, "creating connection to ", dest)
	var conn net.Conn
	if streamSettings.ProtocolSettings.(*Config).Ed > 0 {
		ctx, cancel := context.WithCancel(ctx)
		conn = &delayDialConn{
			dialed:         make(chan bool, 1),
			cancel:         cancel,
			ctx:            ctx,
			dest:           dest,
			streamSettings: streamSettings,
		}
	} else {
		var err error
		if conn, err = dialWebSocket(ctx, dest, streamSettings, nil); err != nil {
			return nil, errors.New("failed to dial WebSocket").Base(err)
		}
	}
	return stat.Connection(conn), nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}

func dialWebSocket(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, ed []byte) (net.Conn, error) {
	wsSettings := streamSettings.ProtocolSettings.(*Config)

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return internet.DialSystem(ctx, dest, streamSettings.SocketSettings)
		},
		ReadBufferSize:   4 * 1024,
		WriteBufferSize:  4 * 1024,
		HandshakeTimeout: time.Second * 8,
	}

	protocol := "ws"

	tConfig := tls.ConfigFromStreamSettings(streamSettings)
	if tConfig != nil {
		protocol = "wss"
		tlsConfig := tConfig.GetTLSConfig(tls.WithDestination(dest), tls.WithNextProto("http/1.1"))
		dialer.TLSClientConfig = tlsConfig
		if fingerprint := tls.GetFingerprint(tConfig.Fingerprint); fingerprint != nil {
			dialer.NetDialTLSContext = func(_ context.Context, _, addr string) (gonet.Conn, error) {
				// Like the NetDial in the dialer
				pconn, err := internet.DialSystem(ctx, dest, streamSettings.SocketSettings)
				if err != nil {
					errors.LogErrorInner(ctx, err, "failed to dial to "+addr)
					return nil, err
				}
				// TLS and apply the handshake
				cn := tls.UClient(pconn, tlsConfig, fingerprint).(*tls.UConn)
				if err := cn.WebsocketHandshakeContext(ctx); err != nil {
					errors.LogErrorInner(ctx, err, "failed to dial to "+addr)
					return nil, err
				}
				if !tlsConfig.InsecureSkipVerify {
					if err := cn.VerifyHostname(tlsConfig.ServerName); err != nil {
						errors.LogErrorInner(ctx, err, "failed to dial to "+addr)
						return nil, err
					}
				}
				return cn, nil
			}
		}
	}

	host := dest.NetAddr()
	if (protocol == "ws" && dest.Port == 80) || (protocol == "wss" && dest.Port == 443) {
		host = dest.Address.String()
	}
	// Randomize the case of the host in the URI
	uri := protocol + "://" + randomizeCase(host) + wsSettings.GetNormalizedPath()

	if browser_dialer.HasBrowserDialer() {
		conn, err := browser_dialer.DialWS(uri, ed)
		if err != nil {
			return nil, err
		}

		return NewConnection(conn, conn.RemoteAddr(), nil, wsSettings.HeartbeatPeriod), nil
	}

	header := wsSettings.GetRequestHeader()
	// See dialer.DialContext()
	header.Set("Host", wsSettings.Host)
	if header.Get("Host") == "" && tConfig != nil {
		header.Set("Host", tConfig.ServerName)
	}
	if header.Get("Host") == "" {
		header.Set("Host", dest.Address.String())
	}
	if ed != nil {
		// RawURLEncoding is support by both V2Ray/V2Fly and XRay.
		header.Set("Sec-WebSocket-Protocol", base64.RawURLEncoding.EncodeToString(ed))
	}

	// ==========================================================
	//  MODIFICATION APPLIED HERE
	// ==========================================================
	// Apply randomization and set User-Agent.
	// This function returns a new header map because header keys are canonicalized.
	header = applyHeaderModifications(header)
	// ==========================================================

	conn, resp, err := dialer.DialContext(ctx, uri, header)
	if err != nil {
		var reason string
		if resp != nil {
			reason = resp.Status
		}
		return nil, errors.New("failed to dial to (", uri, "): ", reason).Base(err)
	}

	return NewConnection(conn, conn.RemoteAddr(), nil, wsSettings.HeartbeatPeriod), nil
}

type delayDialConn struct {
	net.Conn
	closed         bool
	dialed         chan bool
	cancel         context.CancelFunc
	ctx            context.Context
	dest           net.Destination
	streamSettings *internet.MemoryStreamConfig
}

func (d *delayDialConn) Write(b []byte) (int, error) {
	if d.closed {
		return 0, io.ErrClosedPipe
	}
	if d.Conn == nil {
		ed := b
		if len(ed) > int(d.streamSettings.ProtocolSettings.(*Config).Ed) {
			ed = nil
		}
		var err error
		if d.Conn, err = dialWebSocket(d.ctx, d.dest, d.streamSettings, ed); err != nil {
			d.Close()
			return 0, errors.New("failed to dial WebSocket").Base(err)
		}
		d.dialed <- true
		if ed != nil {
			return len(ed), nil
		}
	}
	return d.Conn.Write(b)
}

func (d *delayDialConn) Read(b []byte) (int, error) {
	if d.closed {
		return 0, io.ErrClosedPipe
	}
	if d.Conn == nil {
		select {
		case <-d.ctx.Done():
			return 0, io.ErrUnexpectedEOF
		case <-d.dialed:
		}
	}
	return d.Conn.Read(b)
}

func (d *delayDialConn) Close() error {
	d.closed = true
	d.cancel()
	if d.Conn == nil {
		return nil
	}
	return d.Conn.Close()
}
