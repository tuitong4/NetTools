package ping

type PingSetting struct {
	SourceIP        map[string][]string //ping源地址组
	DefaultIP       string //默认地址，ping时候使用的默认地址
	DefaultNetType  string //默认网络类型
	PingCount       int
	TimeOutMs       int
	WorkInterval    int
	MaxRoutineCount int
	SrcBind         bool
	PingMode        int // 0：不设定ping的源地址；1：均分目标到同一个组下的所有源IP地址上；2：不同组的源IP都去ping相同的目标
}

type PingRawSetting struct {
	SourceIP        string `ini:"source_ip"`//ping源地址组
	DefaultIP       string `ini:"default_ip"`//默认地址，ping时候使用的默认地址
	DefaultNetType  string `ini:"default_net_type"`//默认网络类型
	PingCount       int	`ini:"ping_count"`
	TimeOutMs       int	`ini:"timeout_ms"`
	WorkInterval    int `ini:"work_interval_sec"`
	MaxRoutineCount int `ini:"max_routine_count"`
	SrcBind         bool `ini:"source_bind"`
	PingMode        int `ini:"ping_mode"`// 0：不设定ping的源地址；1：均分目标到同一个组下的所有源IP地址上；2：不同组的源IP都去ping相同的目标
}