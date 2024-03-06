package session

import (
	"fmt"
	"golang.org/x/net/websocket"
)

type Status uint16

const (
	Connected                    Status = 0    //  正常连接。
	CloseNormalClosure           Status = 1000 //  正常关闭连接。
	CloseGoingAway               Status = 1001 //  终端离开，表示终端正在关闭连接，例如浏览器标签被关闭或导航离开当前页面。
	CloseProtocolError           Status = 1002 //  协议错误，表示由于协议错误导致连接关闭。
	CloseUnsupportedData         Status = 1003 //  不支持的数据，表示接收到了不支持的数据类型或格式。
	CloseNoStatusReceived        Status = 1005 //  未接收到状态码，表示连接关闭时未接收到预期的状态码。
	CloseAbnormalClosure         Status = 1006 //  异常关闭，表示连接意外关闭，无法确定具体原因。
	CloseInvalidFramePayloadData Status = 1007 //  无效的帧载荷数据，表示接收到的帧载荷数据不符合协议规范。
	ClosePolicyViolation         Status = 1008 //  协议违规，表示违反了协议的约束或策略。
	CloseMessageTooBig           Status = 1009 //  消息过大，表示接收到的消息超过了允许的最大大小。
	CloseMandatoryExtension      Status = 1010 //  必需 扩展，表示服务器要求使用一个或多个扩展，但客户端未提供。
	CloseInternalServerErr       Status = 1011 //  内部服务器错误，表示服务器在处理连接时遇到了内部错误。
	CloseServiceRestart          Status = 1012 //  服务重启，表示服务器正在重新启动。
	CloseTryAgainLater           Status = 1013 //  请稍后重试，表示服务器暂时无法处理连接，请求客户端稍后再试。
	CloseTLSHandshake            Status = 1015 //  TLS握手错误，表示TLS握手过程中发生错误。
)

func StatusToError(s Status) error {
	switch s {
	case Connected:
		return nil
	case CloseNormalClosure:
		return nil
	case CloseGoingAway:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 终端离开，表示终端正在关闭连接，例如浏览器标签被关闭或导航离开当前页面", CloseGoingAway),
		}
	case CloseProtocolError:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 协议错误，表示由于协议错误导致连接关闭", CloseProtocolError),
		}
	case CloseUnsupportedData:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 不支持的数据，表示接收到了不支持的数据类型或格式", CloseUnsupportedData),
		}
	case CloseNoStatusReceived:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 未接收到状态码，表示连接关闭时未接收到预期的状态码", CloseNoStatusReceived),
		}
	case CloseAbnormalClosure:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 异常关闭，表示连接意外关闭，无法确定具体原因", CloseAbnormalClosure),
		}
	case CloseInvalidFramePayloadData:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 无效的帧载荷数据，表示接收到的帧载荷数据不符合协议规范", CloseInvalidFramePayloadData),
		}
	case ClosePolicyViolation:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 协议违规，表示违反了协议的约束或策略", CloseInvalidFramePayloadData),
		}
	case CloseMessageTooBig:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 消息过大，表示接收到的消息超过了允许的最大大小", CloseInvalidFramePayloadData),
		}
	case CloseMandatoryExtension:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 必需扩展，表示服务器要求使用一个或多个扩展，但客户端未提供", CloseMandatoryExtension),
		}
	case CloseInternalServerErr:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 内部服务器错误，表示服务器在处理连接时遇到了内部错误", CloseInternalServerErr),
		}
	case CloseServiceRestart:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 服务重启，表示服务器正在重新启动", CloseInternalServerErr),
		}
	case CloseTryAgainLater:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: 请稍后重试，表示服务器暂时无法处理连接，请求客户端稍后再试", CloseTryAgainLater),
		}
	case CloseTLSHandshake:
		return &websocket.ProtocolError{
			ErrorString: fmt.Sprintf("%d: TLS握手错误，表示TLS握手过程中发生错误", CloseTLSHandshake),
		}
	}
	return &websocket.ProtocolError{
		ErrorString: "Unknown errors",
	}
}
