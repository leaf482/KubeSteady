package optimizer

import "time"

type Recommendation struct {
	Workload      string
	Namespace     string
	CPURequestMil int64
	MemoryMiB     int64
	GeneratedAt   time.Time
}
