package ant_schedule

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"
	"k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	"math"
)

const AntSchedulerName = "AntSchedule"

type AntScheduler struct {
	NodeArray           []*nodeinfo.NodeInfo
	PodArray            []*v1.Pod
	availableNodeArray  []*nodeinfo.NodeInfo
	pheromoneMatrix     [][]float64
	MaxPheromoneMap     []int
	unscheduledPods     map[int]int
	criticalPointMatrix []int //在一次迭代中，随机分配策略的蚂蚁临界编号

	iteratorNum         int
	antNum              int
	pheromoneDecayRatio float64
	pheromoneRaiseRatio float64
}

func New(nodeArray []*nodeinfo.NodeInfo, pods []*v1.Pod) *AntScheduler {
	//availableNodes := make([]*v1.Node, len(nodeArray))
	//for i := 0; i < len(nodeArray); i++ {
	//	availableNodes[i] = nodeArray[i].Node()
	//}
	return &AntScheduler{
		NodeArray:           nodeArray,
		PodArray:            pods,
		availableNodeArray:  nodeArray,
		unscheduledPods:     make(map[int]int),
		criticalPointMatrix: make([]int, len(pods)),
		MaxPheromoneMap:     make([]int, len(pods)),
	}
}

func (ant *AntScheduler) WithDecayRatio(decay float64) *AntScheduler {
	ant.pheromoneDecayRatio = decay
	return ant
}

func (ant *AntScheduler) WithRaiseRatio(raise float64) *AntScheduler {
	ant.pheromoneRaiseRatio = raise
	return ant
}

func (ant *AntScheduler) WithIteratorNum(it int) *AntScheduler {
	ant.iteratorNum = it
	return ant
}

func (ant *AntScheduler) WithAntNum(antNum int) *AntScheduler {
	ant.antNum = antNum
	return ant
}

func (ant *AntScheduler) InitPheromoneMatrix() *AntScheduler {
	pm := make([][]float64, len(ant.PodArray))
	for i := 0; i < len(ant.PodArray); i++ {
		pm[i] = make([]float64, len(ant.NodeArray))
		for j := 0; j < len(ant.NodeArray); j++ {
			pm[i][j] = 1.0
		}
	}
	ant.pheromoneMatrix = pm
	return ant
}

func (ant *AntScheduler) assignOnePod(antCount int, podCount int) int {
	pod := ant.PodArray[podCount]
	//计算当前pod的资源需求
	var podCpu int64
	podCpu = noderesources.CalculatePodResourceRequest(pod, v1.ResourceCPU)
	//for _, container := range pod.Spec.Containers {
	//	containerCpu := container.Resources.Requests.Cpu().MilliValue()
	//	podCpu += containerCpu
	//}
	var podMem int64
	podMem = noderesources.CalculatePodResourceRequest(pod, v1.ResourceMemory)
	//for _, container := range pod.Spec.Containers {
	//	containerMem := container.Resources.Requests.Memory().Value()
	//	podMem += containerMem
	//}
	nodeIndex := ant.MaxPheromoneMap[podCount]
	node := ant.availableNodeArray[nodeIndex]
	//计算node的资源
	nodeCpu := node.AllocatableResource().MilliCPU
	nodeMem := node.AllocatableResource().Memory

	if antCount <= ant.criticalPointMatrix[podCount] {
		//当前蚂蚁在临界编号之前，因此根据信息素浓度最大的挑选node
		if nodeCpu >= podCpu && nodeMem >= podMem {
			return nodeIndex
		}
	}

	retryCount := 0
	// 随机重试3次
	for retryCount < 3 && (podCpu > nodeCpu || podMem > nodeMem) {
		nodeIndex := rand.Intn(len(ant.NodeArray))
		node = ant.availableNodeArray[nodeIndex]

		if podCpu <= nodeCpu && podMem <= nodeMem {
			return nodeIndex
		}

		retryCount++
	}

	for i := 0; i < len(ant.NodeArray); i++ {
		node = ant.availableNodeArray[i]

		if podCpu <= nodeCpu && podMem <= nodeMem {
			return i
		}
	}

	// 返回0表示没有调度pod到一个node上
	return -1
}

func (ant *AntScheduler) updatePheromoneMatrix(minPathOneAnt []int) {
	// 将所有信息素衰减
	for i := 0; i < len(ant.PodArray); i++ {
		for j := 0; j < len(ant.NodeArray); j++ {
			ant.pheromoneMatrix[i][j] *= ant.pheromoneDecayRatio
		}
	}

	// 将本次迭代中最优路径的信息素增加
	for podIndex := 0; podIndex < len(ant.PodArray); podIndex++ {
		nodeIndex := minPathOneAnt[podIndex]
		ant.pheromoneMatrix[podIndex][nodeIndex] *= ant.pheromoneRaiseRatio
	}

	for podIndex := 0; podIndex < len(ant.PodArray); podIndex++ {
		maxPheromone := ant.pheromoneMatrix[podIndex][0]
		maxIndex := 0
		sumPheromone := ant.pheromoneMatrix[podIndex][0]
		isAllSame := true

		for nodeIndex := 1; nodeIndex < len(ant.NodeArray); nodeIndex++ {
			if ant.pheromoneMatrix[podIndex][nodeIndex] > maxPheromone {
				maxPheromone = ant.pheromoneMatrix[podIndex][nodeIndex]
				maxIndex = nodeIndex
			}

			if ant.pheromoneMatrix[podIndex][nodeIndex] != ant.pheromoneMatrix[podIndex][nodeIndex-1] {
				isAllSame = false
			}

			sumPheromone += ant.pheromoneMatrix[podIndex][nodeIndex]
		}

		if isAllSame == true {
			maxIndex = rand.Intn(len(ant.NodeArray))
			maxPheromone = ant.pheromoneMatrix[podIndex][maxIndex]
		}

		//每次更新信息素矩阵后，也要更新pod对应最大的信息素下标，这个就是最后调度的结果
		ant.MaxPheromoneMap[podIndex] = maxIndex

		ant.criticalPointMatrix[podIndex] = int(math.Round(float64(ant.antNum) * (maxPheromone / sumPheromone)))
	}

}

func (ant *AntScheduler) AcaSearch() *AntScheduler {
	for itCount := 0; itCount < ant.iteratorNum; itCount++ {

		minNodeNum := 10000
		var minPathOneAnt []int
		for antCount := 0; antCount < ant.antNum; antCount++ {
			ant.resetAvailableNodeArray()                // 重置可用资源数组
			pathOneAnt := make([]int, len(ant.PodArray)) // 重置当前蚂蚁的路径
			ant.unscheduledPods = make(map[int]int)      // 重置未调度的pod的数组

			hasPodUnscheduled := false
			for podCount := 0; podCount < len(ant.PodArray); podCount++ {
				nodeCount := ant.assignOnePod(antCount, podCount)
				if nodeCount >= 0 {
					pathOneAnt[podCount] = nodeCount
					// 这里是要更新可获取节点的cpu和内存值的，需要对Node对象做修改的
					node := *ant.availableNodeArray[nodeCount]
					//需要从node中把调度的pod的cpu和内存数减掉
					var podCpu int64
					podCpu = noderesources.CalculatePodResourceRequest(ant.PodArray[podCount], v1.ResourceCPU)
					var podMem int64
					podMem = noderesources.CalculatePodResourceRequest(ant.PodArray[podCount], v1.ResourceMemory)

					//更新本地缓存node的内存cpu信息
					nodeCpu := node.AllocatableResource().MilliCPU
					nodeMem := node.AllocatableResource().Memory
					nodeCpu = nodeCpu - podCpu
					nodeMem = nodeMem - podMem
					node.SetAllocatableResource(&nodeinfo.Resource{
						MilliCPU: nodeCpu,
						Memory:   nodeMem,
					})
				} else {
					ant.unscheduledPods[podCount] = -1
					pathOneAnt[podCount] = -1 // -1表示没有调度该pod
					hasPodUnscheduled = true
				}

			}

			// 如果当前路径中有pod没有调度，不参与比较
			if hasPodUnscheduled == false {
				var nodeSet = make(map[int]int) // 使用map实现set
				for i := 0; i < len(ant.PodArray); i++ {
					nodeSet[pathOneAnt[i]] = 0
				}

				if len(nodeSet) < minNodeNum {
					minNodeNum = len(nodeSet)
					minPathOneAnt = pathOneAnt // pathOneAnt中如果含有-1的话，不能将其赋值给minPathOneAnt，否则在后面更新信息素可能会数组越界
				}
			}
		}

		ant.updatePheromoneMatrix(minPathOneAnt)
	}
	return ant
}

func (ant *AntScheduler) resetAvailableNodeArray() {
	copy(ant.availableNodeArray, ant.NodeArray)
}
