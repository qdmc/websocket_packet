package session

import (
	"fmt"
	"golang.org/x/net/websocket"
)

type Status uint16

const (
	Connected          Status = 200  //  正常连接。
	CloseNormalClosure Status = 1000 //  正常关闭连接。

	CloseInvalidFramePayloadData Status = 1007 //  无效的帧载荷数据，表示接收到的帧载荷数据不符合协议规范。
	CloseIHartTimeOut            Status = 2001 //  心跳超时
	CloseReadConnFailed          Status = 2002 //  读取失败
	CloseWriteConnFailed         Status = 2003 //  写入失败
)

func StatusToError(s Status) error {
	switch s {
	case Connected:
		return nil
	case CloseNormalClosure:
		return nil
	case CloseInvalidFramePayloadData:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: Invalid frame payload data", CloseInvalidFramePayloadData),
		}
	case CloseIHartTimeOut:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: Heartbeat Timeout,close connection", CloseIHartTimeOut),
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
