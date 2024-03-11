package session

type ConfigureSession struct {
	ConnectedCallBackHandle ConnectedCallBackHandle  // 建立链接后的回调
	DisConnectCallBack      DisConnectCallBackHandle // 断开链接后的回调
	FrameCallBackHandle     FrameCallBackHandle      // 帧读取后的回调
	IsStatistics            bool                     // 是否开启流量统计,默认为false
	AutoPingTicker          int64                    // 自动发送pingFrame的时间(秒)配置, <1:关闭(默认值); 1~~25:都会配置为25秒; >120:都会配置为120秒
}
