package session

import (
	"github.com/qdmc/websocket_packet/frame"
)

// Status        session状态
type Status = frame.CloseStatus

const (
	ClientCreate         = frame.SessionClientCreate    //  客户端建立
	ClientReconnect      = frame.SessionClientReconnect //  客户端重新链接
	Connected            = frame.SessionConnected       //  正常连接。
	ClientConnectFailed  = frame.SessionConnectFailed   // 链接失败
	CloseNormalClosure   = frame.CloseNormalClosure     // 正常关闭连接
	CloseHartTimeOut     = frame.ClosePolicyViolation   //  心跳超时
	CloseWriteConnFailed = frame.CloseGoingAway         //  写入失败
	CloseReadConnFailed  = frame.CloseGoingAway         // 读取失败
)
