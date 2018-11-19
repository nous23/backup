package values

import "time"

const (
	RobocopyRetryCount int = 5
	MdRetryCount int = 1
	MonitConfigPeriod = time.Second * 5
	RecentRecordCount int = 32
)
