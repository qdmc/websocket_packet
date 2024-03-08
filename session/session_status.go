package session

import (
	"errors"
	"fmt"
)

// Status        session状态
type Status byte

const (
	ClientCreate         Status = 0 //  客户端建立
	ClientReconnect      Status = 1 //  客户端重新链接
	Connected            Status = 2 //  正常连接。
	CloseNormalClosure   Status = 3 //  正常关闭连接。
	CloseHartTimeOut     Status = 4 //  心跳超时
	CloseReadConnFailed  Status = 5 //  读取失败
	CloseWriteConnFailed Status = 6 //  写入失败
)

// StatusToError    状态转换成 error
func StatusToError(s Status) error {
	switch s {
	case Connected:
		return nil
	case CloseNormalClosure:
		return nil
	case CloseHartTimeOut:
		return errors.New(fmt.Sprintf("%d: Heartbeat Timeout,close connection", CloseHartTimeOut))
	case CloseReadConnFailed:
		return errors.New(fmt.Sprintf("%d: Read connection failed,close connection", CloseReadConnFailed))
	case CloseWriteConnFailed:
		return errors.New(fmt.Sprintf("%d: Write to connection failed,close connection", CloseWriteConnFailed))
	}
	return errors.New("unknown errors")
}
