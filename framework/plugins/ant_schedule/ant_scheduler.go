package ant_schedule

import v1 "k8s.io/api/core/v1"

const AntSchedulerName = "AntSchedule"

var (
	podArray = make([]v1.Pod, 0)
	nodeArray = make([]v1.Node, 0)
)
