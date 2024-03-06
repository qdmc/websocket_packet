package uity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
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
		return errors.New("request method is not GET")
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Connection", "upgrade") {
		return errors.New("not found in 'Connection' header")
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Upgrade", "websocket") {
		return errors.New("not found in 'Upgrade' header")
	}
	if !CheckHttpHeaderKeyVale(r.Header, "Sec-Websocket-Version", "13") {
		return errors.New("unsupported version: 13 not found in 'Sec-Websocket-Version' header")
	}
	if !checkSecWebsocketKey(r.Header) {
		return errors.New("not a websocket handshake: 'Sec-WebSocket-Key' header must be Base64 encoded value of 16-byte in length")
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
	//protocol := req.Header.Get("Sec-Websocket-Protocol")
	//extensions := req.Header.Get("Sec-Websocket-Extensions")
	key := req.Header.Get("Sec-Websocket-Key")
	var p []byte
	p = append(p, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "...)
	p = append(p, ComputeAcceptKey(key)...)
	p = append(p, "\r\n"...)
	//if protocol != "" {
	//	p = append(p, "Sec-WebSocket-Protocol: "...)
	//	p = append(p, protocol...)
	//	p = append(p, "\r\n"...)
	//}
	//if extensions != "" {
	//	p = append(p, "Sec-WebSocket-Extensions: "...)
	//	p = append(p, extensions...)
	//	p = append(p, "\r\n"...)
	//}
	//p = append(p, "Sec-WebSocket-Version: "...)
	//p = append(p, "13"...)
	//p = append(p, "\r\n"...)
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
