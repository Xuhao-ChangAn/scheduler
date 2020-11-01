package ant_schedule

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	"math"
)

const AntSchedulerName = "AntSchedule"

type AntScheduler struct {
	NodeArray           []*v1.Node
	PodArray            []*v1.Pod
	availableNodeArray  []*v1.Node
	pheromoneMatrix     [][]float64
	maxPheromoneMap     []int
	unscheduledPods     map[int]int
	criticalPointMatrix []int //在一次迭代中，随机分配策略的蚂蚁临界编号
	iteratorNum         int
	antNum              int

	pheromoneDecayRatio float64
	pheromoneRaiseRatio float64
}

func New(nodeArray []*nodeinfo.NodeInfo, pods []*v1.Pod) *AntScheduler {
	availableNodes := make([]*v1.Node, len(nodeArray))
	for i := 0; i < len(nodeArray); i++ {
		availableNodes[i] = nodeArray[i].Node()
	}
	return &AntScheduler{
		NodeArray:          availableNodes,
		PodArray:           pods,
		availableNodeArray: availableNodes,
		unscheduledPods:    make(map[int]int),
	}
}

//TODO 可以考虑定义函数类型
func (ant *AntScheduler) withDecayRatio(decay float64) *AntScheduler {
	ant.pheromoneDecayRatio = decay
	return ant
}

func (ant *AntScheduler) withRaiseRatio(raise float64) *AntScheduler {
	ant.pheromoneRaiseRatio = raise
	return ant
}

func (ant *AntScheduler) withIteratorNum(it int) *AntScheduler {
	ant.iteratorNum = it
	return ant
}

func (ant *AntScheduler) withAntNum(antNum int) *AntScheduler {
	ant.antNum = antNum
	return ant
}

func (ant *AntScheduler) initPheromoneMatrix() *AntScheduler {
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
	//TODO 需要对返回的Resource类型做一定的解析
	var podCpu int64
	for _, container := range pod.Spec.Containers {
		containerCpu := container.Resources.Requests.Cpu().Value()
		podCpu += containerCpu
	}
	var podMem int64
	for _, container := range pod.Spec.Containers {
		containerMem := container.Resources.Requests.Memory().Value()
		podMem += containerMem
	}

	if antCount <= ant.criticalPointMatrix[podCount] {
		//当前蚂蚁在临界编号之前，因此根据信息素浓度最大的挑选node
		nodeIndex := ant.maxPheromoneMap[podCount]
		node := &ant.availableNodeArray[nodeIndex]
		//计算node的资源
		nodeCpu := (*node).Status.Capacity.Cpu().Value()
		nodeMem := (*node).Status.Capacity.Memory().Value()

		if nodeCpu >= podCpu && nodeMem >= podMem {
			return nodeIndex
		}
	}

	nodeIndex := rand.Intn(len(ant.NodeArray))
	node := &ant.availableNodeArray[nodeIndex]
	nodeCpu := (*node).Status.Capacity.Cpu().Value()
	nodeMem := (*node).Status.Capacity.Memory().Value()
	retryCount := 0
	// 随机重试3次
	for retryCount < 3 && (podCpu > nodeCpu || podMem > nodeMem) {
		nodeIndex := rand.Intn(len(ant.NodeArray))
		node = &ant.availableNodeArray[nodeIndex]

		if podCpu <= nodeCpu && podMem <= nodeMem {
			return nodeIndex
		}

		retryCount++
	}

	for i := 0; i < len(ant.NodeArray); i++ {
		node = &ant.availableNodeArray[i]

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

		ant.maxPheromoneMap[podIndex] = maxIndex

		ant.criticalPointMatrix[podIndex] = int(math.Round(float64(ant.antNum) * (maxPheromone / sumPheromone)))
	}

}

func (ant *AntScheduler) acaSearch() {
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
					//TODO 这里是要更新可获取节点的cpu和内存值的，需要对Node对象做修改的
					node := ant.availableNodeArray[nodeCount]
					//需要从node中把调度的pod的cpu和内存数减掉
					var podCpu int64
					for _, podContainer := range ant.PodArray[podCount].Spec.Containers {
						containerCpu := podContainer.Resources.Requests.Cpu().Value()
						podCpu += containerCpu
					}
					var podMem int64
					for _, podContainer := range ant.PodArray[podCount].Spec.Containers {
						containerMem := podContainer.Resources.Requests.Memory().Value()
						podMem += containerMem
					}

					nodeCpu := (*node).Status.Capacity.Cpu().Value()
					nodeMem := (*node).Status.Capacity.Memory().Value()
					nodeCpu = nodeCpu - podCpu
					nodeMem = nodeMem - podMem
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

		fmt.Println("第", itCount+1, "轮最小机器数:", minNodeNum)

		ant.updatePheromoneMatrix(minPathOneAnt)
	}
}

func (ant *AntScheduler) resetAvailableNodeArray() {
	copy(ant.availableNodeArray, ant.NodeArray)
}
