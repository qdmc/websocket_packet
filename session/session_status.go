package session

import (
	"fmt"
	"golang.org/x/net/websocket"
)

type Status byte

const (
	ClientCreate         Status = 0
	ClientReconnect      Status = 1
	Connected            Status = 2 //  正常连接。
	CloseNormalClosure   Status = 3 //  正常关闭连接。
	CloseHartTimeOut     Status = 4 //  心跳超时
	CloseReadConnFailed  Status = 5 //  读取失败
	CloseWriteConnFailed Status = 6 //  写入失败
)

func StatusToError(s Status) error {
	switch s {
	case Connected:
		return nil
	case CloseNormalClosure:
		return nil
	case CloseHartTimeOut:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: Heartbeat Timeout,close connection", CloseHartTimeOut),
		}
	case CloseReadConnFailed:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: Read connection failed,close connection", CloseReadConnFailed),
		}
	case CloseWriteConnFailed:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: Write to connection failed,close connection", CloseWriteConnFailed),
		}
	}
	return &websocket.ProtocolError{
		ErrorString: "Unknown errors",
	}
}
