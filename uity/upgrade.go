package uity

import (
	"crypto/rand"
	"encoding/base64"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func DefaultUpgradeCheckHandle(r *http.Request) ([]byte, error) {
	err := DefaultUpgradeCheck(r)
	if err != nil {
		return nil, err
	}
	return MakeServerHandshakeBytes(r), nil
}

func DefaultUpgradeCheck(r *http.Request) error {
	if r.Method != http.MethodGet {
		return websocket.ErrBadRequestMethod
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Connection", "upgrade") {
		return websocket.ErrBadUpgrade
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Upgrade", "websocket") {
		return websocket.ErrBadWebSocketProtocol
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Sec-Websocket-Version", "13") {
		return websocket.ErrBadProtocolVersion
	}
	if !checkSecWebsocketKey(r.Header) {
		return &websocket.ProtocolError{ErrorString: "not a websocket handshake: 'Sec-WebSocket-Key' header must be Base64 encoded value of 16-byte in length"}
	}
	return nil
}

func CheckHttpHeaderValue(h http.Header, key string) ([]string, bool) {
	if key == "" {
		return nil, false
	}
	if val, ok := h[key]; ok {
		return val, true
	} else {
		return nil, false
	}
}

func CheckHttpHeaderKeyVale(h http.Header, key, value string) bool {
	if vals, ok := CheckHttpHeaderValue(h, key); ok {
		valueStr := strings.ToLower(value)
		for _, val := range vals {
			if strings.Index(strings.ToLower(val), valueStr) != -1 {
				return true
			}
		}
		return false
	} else {
		return false
	}
}

func checkSecWebsocketKey(h http.Header) bool {
	key := h.Get("Sec-Websocket-Key")
	if key == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(decoded) == 16
}

func MakeServerHandshakeBytes(req *http.Request) []byte {
	key := req.Header.Get("Sec-Websocket-Key")
	var p []byte
	p = append(p, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "...)
	p = append(p, ComputeAcceptKey(key)...)
	p = append(p, "\r\n"...)
	p = append(p, "\r\n"...)
	return p
}

// HttpResponseError    Response回复错误,在拆解Response前使用
func HttpResponseError(w http.ResponseWriter, status int, err error) {
	errStr := http.StatusText(status)
	if err != nil && err.Error() != "" {
		errStr = err.Error()
	}
	http.Error(w, errStr, status)
}

// GenerateChallengeKey   生成随机的websocketKey
func GenerateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

func HostPortNoPort(u *url.URL) (hostPort, hostNoPort string) {
	hostPort = u.Host
	hostNoPort = u.Host
	if i := strings.LastIndex(u.Host, ":"); i > strings.LastIndex(u.Host, "]") {
		hostNoPort = hostNoPort[:i]
	} else {
		switch u.Scheme {
		case "wss":
			hostPort += ":443"
		case "https":
			hostPort += ":443"
		default:
			hostPort += ":80"
		}
	}
	return hostPort, hostNoPort
}
