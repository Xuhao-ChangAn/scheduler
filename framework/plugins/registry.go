/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugins

import (
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultpodtopologyspread"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/imagelocality"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/interpodaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodelabel"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodename"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeports"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodepreferavoidpods"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodevolumelimits"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/serviceaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumebinding"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumerestrictions"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumezone"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

// NewInTreeRegistry builds the registry with all the in-tree plugins.
// A scheduler that runs out of tree plugins can register additional plugins
// through the WithFrameworkOutOfTreeRegistry option.
func NewInTreeRegistry() framework.Registry {
	return framework.Registry{
		//主要实现了打分的插件，用于拓扑感知，负载平衡用的
		defaultpodtopologyspread.Name:              defaultpodtopologyspread.New,
		//打分插件，用户镜像感知
		imagelocality.Name:                         imagelocality.New,
		//过滤和打分插件
		tainttoleration.Name:                       tainttoleration.New,
		//过滤，主要是根据node名字进行过滤的
		nodename.Name:                              nodename.New,
		//过滤，查看当前期望的端口，主机是否拥有
		nodeports.Name:                             nodeports.New,
		//打分，node标签选择
		nodepreferavoidpods.Name:                   nodepreferavoidpods.New,
		//过滤和打分，nodeSelector不match的过滤掉，符合的按照满足的程度进行加分
		nodeaffinity.Name:                          nodeaffinity.New,
		//pod拓扑的问题，没有插件的接口实现
		podtopologyspread.Name:                     podtopologyspread.New,
		//过滤插件，主要是看pod是否容忍node的unScheduable标签
		nodeunschedulable.Name:                     nodeunschedulable.New,
		//检查一个node是否有足够的资源
		noderesources.FitName:                      noderesources.NewFit,
		//会计算CPU和内存的比值，如果不在0-1之间就不符合
		noderesources.BalancedAllocationName:       noderesources.NewBalancedAllocation,
		//资源被占用的越多，分值越高
		noderesources.MostAllocatedName:            noderesources.NewMostAllocated,
		//计算未使用资源的情况，未使用的越多，则分值越高
		noderesources.LeastAllocatedName:           noderesources.NewLeastAllocated,

		noderesources.RequestedToCapacityRatioName: noderesources.NewRequestedToCapacityRatio,
		noderesources.ResourceLimitsName:           noderesources.NewResourceLimits,
		volumebinding.Name:                         volumebinding.New,
		volumerestrictions.Name:                    volumerestrictions.New,
		volumezone.Name:                            volumezone.New,
		nodevolumelimits.CSIName:                   nodevolumelimits.NewCSI,
		nodevolumelimits.EBSName:                   nodevolumelimits.NewEBS,
		nodevolumelimits.GCEPDName:                 nodevolumelimits.NewGCEPD,
		nodevolumelimits.AzureDiskName:             nodevolumelimits.NewAzureDisk,
		nodevolumelimits.CinderName:                nodevolumelimits.NewCinder,
		interpodaffinity.Name:                      interpodaffinity.New,
		nodelabel.Name:                             nodelabel.New,
		serviceaffinity.Name:                       serviceaffinity.New,
		queuesort.Name:                             queuesort.New,
		defaultbinder.Name:                         defaultbinder.New,
	}
}
