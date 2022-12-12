package costmodel

import (
	"fmt"
	"time"

	"github.com/opencost/opencost/pkg/util/timeutil"

	"github.com/opencost/opencost/pkg/env"
	"github.com/opencost/opencost/pkg/kubecost"
	"github.com/opencost/opencost/pkg/log"
	"github.com/opencost/opencost/pkg/prom"
)

const (
	queryFmtPods                     = `avg(kube_pod_container_status_running{}) by (pod, namespace, %s)[%s:%s]`
	queryFmtPodsUID                  = `avg(kube_pod_container_status_running{}) by (pod, namespace, uid, %s)[%s:%s]`
	queryFmtRAMBytesAllocated        = `avg(avg_over_time(container_memory_allocation_bytes{container!="", container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s, provider_id)`
	queryFmtRAMRequests              = `avg(avg_over_time(kube_pod_container_resource_requests{resource="memory", unit="byte", container!="", container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s)`
	queryFmtRAMUsageAvg              = `avg(avg_over_time(container_memory_working_set_bytes{container!="", container_name!="POD", container!="POD"}[%s])) by (container_name, container, pod_name, pod, namespace, instance, %s)`
	queryFmtRAMUsageMax              = `max(max_over_time(container_memory_working_set_bytes{container!="", container_name!="POD", container!="POD"}[%s])) by (container_name, container, pod_name, pod, namespace, instance, %s)`
	queryFmtCPUCoresAllocated        = `avg(avg_over_time(container_cpu_allocation{container!="", container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s)`
	queryFmtCPURequests              = `avg(avg_over_time(kube_pod_container_resource_requests{resource="cpu", unit="core", container!="", container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s)`
	queryFmtCPUUsageAvg              = `avg(rate(container_cpu_usage_seconds_total{container!="", container_name!="POD", container!="POD"}[%s])) by (container_name, container, pod_name, pod, namespace, instance, %s)`
	queryFmtCPUUsageMax              = `max(rate(container_cpu_usage_seconds_total{container!="", container_name!="POD", container!="POD"}[%s])) by (container_name, container, pod_name, pod, namespace, instance, %s)`
	queryFmtGPUsRequested            = `avg(avg_over_time(kube_pod_container_resource_requests{resource="nvidia_com_gpu", container!="",container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s)`
	queryFmtGPUsAllocated            = `avg(avg_over_time(container_gpu_allocation{container!="", container!="POD", node!=""}[%s])) by (container, pod, namespace, node, %s)`
	queryFmtNodeCostPerCPUHr         = `avg(avg_over_time(node_cpu_hourly_cost[%s])) by (node, %s, instance_type, provider_id)`
	queryFmtNodeCostPerRAMGiBHr      = `avg(avg_over_time(node_ram_hourly_cost[%s])) by (node, %s, instance_type, provider_id)`
	queryFmtNodeCostPerGPUHr         = `avg(avg_over_time(node_gpu_hourly_cost[%s])) by (node, %s, instance_type, provider_id)`
	queryFmtNodeIsSpot               = `avg_over_time(kubecost_node_is_spot[%s])`
	queryFmtPVCInfo                  = `avg(kube_persistentvolumeclaim_info{volumename != ""}) by (persistentvolumeclaim, storageclass, volumename, namespace, %s)[%s:%s]`
	queryFmtPodPVCAllocation         = `avg(avg_over_time(pod_pvc_allocation[%s])) by (persistentvolume, persistentvolumeclaim, pod, namespace, %s)`
	queryFmtPVCBytesRequested        = `avg(avg_over_time(kube_persistentvolumeclaim_resource_requests_storage_bytes{}[%s])) by (persistentvolumeclaim, namespace, %s)`
	queryFmtPVActiveMins             = `count(kube_persistentvolume_capacity_bytes) by (persistentvolume, %s)[%s:%s]`
	queryFmtPVBytes                  = `avg(avg_over_time(kube_persistentvolume_capacity_bytes[%s])) by (persistentvolume, %s)`
	queryFmtPVCostPerGiBHour         = `avg(avg_over_time(pv_hourly_cost[%s])) by (volumename, %s)`
	queryFmtNetZoneGiB               = `sum(increase(kubecost_pod_network_egress_bytes_total{internet="false", sameZone="false", sameRegion="true"}[%s])) by (pod_name, namespace, %s) / 1024 / 1024 / 1024`
	queryFmtNetZoneCostPerGiB        = `avg(avg_over_time(kubecost_network_zone_egress_cost{}[%s])) by (%s)`
	queryFmtNetRegionGiB             = `sum(increase(kubecost_pod_network_egress_bytes_total{internet="false", sameZone="false", sameRegion="false"}[%s])) by (pod_name, namespace, %s) / 1024 / 1024 / 1024`
	queryFmtNetRegionCostPerGiB      = `avg(avg_over_time(kubecost_network_region_egress_cost{}[%s])) by (%s)`
	queryFmtNetInternetGiB           = `sum(increase(kubecost_pod_network_egress_bytes_total{internet="true"}[%s])) by (pod_name, namespace, %s) / 1024 / 1024 / 1024`
	queryFmtNetInternetCostPerGiB    = `avg(avg_over_time(kubecost_network_internet_egress_cost{}[%s])) by (%s)`
	queryFmtNetReceiveBytes          = `sum(increase(container_network_receive_bytes_total{pod!="", container="POD"}[%s])) by (pod_name, pod, namespace, %s)`
	queryFmtNetTransferBytes         = `sum(increase(container_network_transmit_bytes_total{pod!="", container="POD"}[%s])) by (pod_name, pod, namespace, %s)`
	queryFmtNamespaceLabels          = `avg_over_time(kube_namespace_labels[%s])`
	queryFmtNamespaceAnnotations     = `avg_over_time(kube_namespace_annotations[%s])`
	queryFmtPodLabels                = `avg_over_time(kube_pod_labels[%s])`
	queryFmtPodAnnotations           = `avg_over_time(kube_pod_annotations[%s])`
	queryFmtServiceLabels            = `avg_over_time(service_selector_labels[%s])`
	queryFmtDeploymentLabels         = `avg_over_time(deployment_match_labels[%s])`
	queryFmtStatefulSetLabels        = `avg_over_time(statefulSet_match_labels[%s])`
	queryFmtDaemonSetLabels          = `sum(avg_over_time(kube_pod_owner{owner_kind="DaemonSet"}[%s])) by (pod, owner_name, namespace, %s)`
	queryFmtJobLabels                = `sum(avg_over_time(kube_pod_owner{owner_kind="Job"}[%s])) by (pod, owner_name, namespace ,%s)`
	queryFmtPodsWithReplicaSetOwner  = `sum(avg_over_time(kube_pod_owner{owner_kind="ReplicaSet"}[%s])) by (pod, owner_name, namespace ,%s)`
	queryFmtReplicaSetsWithoutOwners = `avg(avg_over_time(kube_replicaset_owner{owner_kind="<none>", owner_name="<none>"}[%s])) by (replicaset, namespace, %s)`
	queryFmtLBCostPerHr              = `avg(avg_over_time(kubecost_load_balancer_cost[%s])) by (namespace, service_name, %s)`
	queryFmtLBActiveMins             = `count(kubecost_load_balancer_cost) by (namespace, service_name, %s)[%s:%s]`
)

// CanCompute should return true if CostModel can act as a valid source for the
// given time range. In the case of CostModel we want to attempt to compute as
// long as the range starts in the past. If the CostModel ends up not having
// data to match, that's okay, and should be communicated with an error
// response from ComputeAllocation.
func (cm *CostModel) CanCompute(start, end time.Time) bool {
	return start.Before(time.Now())
}

// Name returns the name of the Source
func (cm *CostModel) Name() string {
	return "CostModel"
}

// ComputeAllocation uses the CostModel instance to compute an AllocationSet
// for the window defined by the given start and end times. The Allocations
// returned are unaggregated (i.e. down to the container level).
func (cm *CostModel) ComputeAllocation(start, end time.Time, resolution time.Duration) (*kubecost.AllocationSet, error) {
	// If the duration is short enough, compute the AllocationSet directly
	if end.Sub(start) <= cm.MaxPrometheusQueryDuration {
		return cm.computeAllocation(start, end, resolution)
	}

	// If the duration exceeds the configured MaxPrometheusQueryDuration, then
	// query for maximum-sized AllocationSets, collect them, and accumulate.

	// s and e track the coverage of the entire given window over multiple
	// internal queries.
	s, e := start, start

	// Collect AllocationSets in a range, then accumulate
	// TODO optimize by collecting consecutive AllocationSets, accumulating as we go
	asr := kubecost.NewAllocationSetRange()

	for e.Before(end) {
		// By default, query for the full remaining duration. But do not let
		// any individual query duration exceed the configured max Prometheus
		// query duration.
		duration := end.Sub(e)
		if duration > cm.MaxPrometheusQueryDuration {
			duration = cm.MaxPrometheusQueryDuration
		}

		// Set start and end parameters (s, e) for next individual computation.
		e = s.Add(duration)

		// Compute the individual AllocationSet for just (s, e)
		as, err := cm.computeAllocation(s, e, resolution)
		if err != nil {
			return kubecost.NewAllocationSet(start, end), fmt.Errorf("error computing allocation for %s: %s", kubecost.NewClosedWindow(s, e), err)
		}

		// Append to the range
		asr.Append(as)

		// Set s equal to e to set up the next query, if one exists.
		s = e
	}

	// Populate annotations, labels, and services on each Allocation. This is
	// necessary because Properties.Intersection does not propagate any values
	// stored in maps or slices for performance reasons. In this case, however,
	// it is both acceptable and necessary to do so.
	allocationAnnotations := map[string]map[string]string{}
	allocationLabels := map[string]map[string]string{}
	allocationServices := map[string]map[string]bool{}

	// Also record errors and warnings, then append them to the results later.
	errors := []string{}
	warnings := []string{}

	for _, as := range asr.Allocations {
		for k, a := range as.Allocations {
			if len(a.Properties.Annotations) > 0 {
				if _, ok := allocationAnnotations[k]; !ok {
					allocationAnnotations[k] = map[string]string{}
				}
				for name, val := range a.Properties.Annotations {
					allocationAnnotations[k][name] = val
				}
			}

			if len(a.Properties.Labels) > 0 {
				if _, ok := allocationLabels[k]; !ok {
					allocationLabels[k] = map[string]string{}
				}
				for name, val := range a.Properties.Labels {
					allocationLabels[k][name] = val
				}
			}

			if len(a.Properties.Services) > 0 {
				if _, ok := allocationServices[k]; !ok {
					allocationServices[k] = map[string]bool{}
				}
				for _, val := range a.Properties.Services {
					allocationServices[k][val] = true
				}
			}
		}

		errors = append(errors, as.Errors...)
		warnings = append(warnings, as.Warnings...)
	}

	// Accumulate to yield the result AllocationSet. After this step, we will
	// be nearly complete, but without the raw allocation data, which must be
	// recomputed.
	result, err := asr.Accumulate()
	if err != nil {
		return kubecost.NewAllocationSet(start, end), fmt.Errorf("error accumulating data for %s: %s", kubecost.NewClosedWindow(s, e), err)
	}

	// Apply the annotations, labels, and services to the post-accumulation
	// results. (See above for why this is necessary.)
	for k, a := range result.Allocations {
		if annotations, ok := allocationAnnotations[k]; ok {
			a.Properties.Annotations = annotations
		}

		if labels, ok := allocationLabels[k]; ok {
			a.Properties.Labels = labels
		}

		if services, ok := allocationServices[k]; ok {
			a.Properties.Services = []string{}
			for s := range services {
				a.Properties.Services = append(a.Properties.Services, s)
			}
		}

		// Expand the Window of all Allocations within the AllocationSet
		// to match the Window of the AllocationSet, which gets expanded
		// at the end of this function.
		a.Window = a.Window.ExpandStart(start).ExpandEnd(end)
	}

	// Maintain RAM and CPU max usage values by iterating over the range,
	// computing maximums on a rolling basis, and setting on the result set.
	for _, as := range asr.Allocations {
		for key, alloc := range as.Allocations {
			resultAlloc := result.Get(key)
			if resultAlloc == nil {
				continue
			}

			if resultAlloc.RawAllocationOnly == nil {
				resultAlloc.RawAllocationOnly = &kubecost.RawAllocationOnlyData{}
			}

			if alloc.RawAllocationOnly == nil {
				// This will happen inevitably for unmounted disks, but should
				// ideally not happen for any allocation with CPU and RAM data.
				if !alloc.IsUnmounted() {
					log.DedupedWarningf(10, "ComputeAllocation: raw allocation data missing for %s", key)
				}
				continue
			}

			if alloc.RawAllocationOnly.CPUCoreUsageMax > resultAlloc.RawAllocationOnly.CPUCoreUsageMax {
				resultAlloc.RawAllocationOnly.CPUCoreUsageMax = alloc.RawAllocationOnly.CPUCoreUsageMax
			}

			if alloc.RawAllocationOnly.RAMBytesUsageMax > resultAlloc.RawAllocationOnly.RAMBytesUsageMax {
				resultAlloc.RawAllocationOnly.RAMBytesUsageMax = alloc.RawAllocationOnly.RAMBytesUsageMax
			}
		}
	}

	// Expand the window to match the queried time range.
	result.Window = result.Window.ExpandStart(start).ExpandEnd(end)

	// Append errors and warnings
	result.Errors = errors
	result.Warnings = warnings

	return result, nil
}

func (cm *CostModel) computeAllocation(start, end time.Time, resolution time.Duration) (*kubecost.AllocationSet, error) {
	// 1. Build out Pod map from resolution-tuned, batched Pod start/end query
	// 2. Run and apply the results of the remaining queries to
	// 3. Build out AllocationSet from completed Pod map

	// Create a window spanning the requested query
	window := kubecost.NewWindow(&start, &end)

	// Create an empty AllocationSet. For safety, in the case of an error, we
	// should prefer to return this empty set with the error. (In the case of
	// no error, of course we populate the set and return it.)
	allocSet := kubecost.NewAllocationSet(start, end)

	// (1) Build out Pod map

	// Build out a map of Allocations as a mapping from pod-to-container-to-
	// underlying-Allocation instance, starting with (start, end) so that we
	// begin with minutes, from which we compute resource allocation and cost
	// totals from measured rate data.
	podMap := map[podKey]*pod{}

	// clusterStarts and clusterEnds record the earliest start and latest end
	// times, respectively, on a cluster-basis. These are used for unmounted
	// PVs and other "virtual" Allocations so that minutes are maximally
	// accurate during start-up or spin-down of a cluster
	clusterStart := map[string]time.Time{}
	clusterEnd := map[string]time.Time{}

	// If ingesting pod UID, we query kube_pod_container_status_running avg
	// by uid as well as the default values, and all podKeys/pods have their
	// names changed to "<pod_name> <pod_uid>". Because other metrics need
	// to generate keys to match pods but don't have UIDs, podUIDKeyMap
	// stores values of format:

	// default podKey : []{edited podkey 1, edited podkey 2}

	// This is because ingesting UID allows us to catch uncontrolled pods
	// with the same names. However, this will lead to a many-to-one metric
	// to podKey relation, so this map allows us to map the metric's
	// "<pod_name>" key to the edited "<pod_name> <pod_uid>" keys in podMap.
	ingestPodUID := env.IsIngestingPodUID()
	podUIDKeyMap := make(map[podKey][]podKey)

	if ingestPodUID {
		log.Debugf("CostModel.ComputeAllocation: ingesting UID data from KSM metrics...")
	}

	// TODO:CLEANUP remove "max batch" idea and clusterStart/End
	err := cm.buildPodMap(window, resolution, env.GetETLMaxPrometheusQueryDuration(), podMap, clusterStart, clusterEnd, ingestPodUID, podUIDKeyMap)
	if err != nil {
		log.Errorf("CostModel.ComputeAllocation: failed to build pod map: %s", err.Error())
	}
	// (2) Run and apply remaining queries

	// Query for the duration between start and end
	durStr := timeutil.DurationString(end.Sub(start))
	if durStr == "" {
		return allocSet, fmt.Errorf("illegal duration value for %s", kubecost.NewClosedWindow(start, end))
	}

	// Convert resolution duration to a query-ready string
	resStr := timeutil.DurationString(resolution)

	ctx := prom.NewNamedContext(cm.PrometheusClient, prom.AllocationContextName)

	queryRAMBytesAllocated := fmt.Sprintf(queryFmtRAMBytesAllocated, durStr, env.GetPromClusterLabel())
	resChRAMBytesAllocated := ctx.QueryAtTime(queryRAMBytesAllocated, end)

	queryRAMRequests := fmt.Sprintf(queryFmtRAMRequests, durStr, env.GetPromClusterLabel())
	resChRAMRequests := ctx.QueryAtTime(queryRAMRequests, end)

	queryRAMUsageAvg := fmt.Sprintf(queryFmtRAMUsageAvg, durStr, env.GetPromClusterLabel())
	resChRAMUsageAvg := ctx.QueryAtTime(queryRAMUsageAvg, end)

	queryRAMUsageMax := fmt.Sprintf(queryFmtRAMUsageMax, durStr, env.GetPromClusterLabel())
	resChRAMUsageMax := ctx.QueryAtTime(queryRAMUsageMax, end)

	queryCPUCoresAllocated := fmt.Sprintf(queryFmtCPUCoresAllocated, durStr, env.GetPromClusterLabel())
	resChCPUCoresAllocated := ctx.QueryAtTime(queryCPUCoresAllocated, end)

	queryCPURequests := fmt.Sprintf(queryFmtCPURequests, durStr, env.GetPromClusterLabel())
	resChCPURequests := ctx.QueryAtTime(queryCPURequests, end)

	queryCPUUsageAvg := fmt.Sprintf(queryFmtCPUUsageAvg, durStr, env.GetPromClusterLabel())
	resChCPUUsageAvg := ctx.QueryAtTime(queryCPUUsageAvg, end)

	queryCPUUsageMax := fmt.Sprintf(queryFmtCPUUsageMax, durStr, env.GetPromClusterLabel())
	resChCPUUsageMax := ctx.QueryAtTime(queryCPUUsageMax, end)

	queryGPUsRequested := fmt.Sprintf(queryFmtGPUsRequested, durStr, env.GetPromClusterLabel())
	resChGPUsRequested := ctx.QueryAtTime(queryGPUsRequested, end)

	queryGPUsAllocated := fmt.Sprintf(queryFmtGPUsAllocated, durStr, env.GetPromClusterLabel())
	resChGPUsAllocated := ctx.QueryAtTime(queryGPUsAllocated, end)

	queryNodeCostPerCPUHr := fmt.Sprintf(queryFmtNodeCostPerCPUHr, durStr, env.GetPromClusterLabel())
	resChNodeCostPerCPUHr := ctx.QueryAtTime(queryNodeCostPerCPUHr, end)

	queryNodeCostPerRAMGiBHr := fmt.Sprintf(queryFmtNodeCostPerRAMGiBHr, durStr, env.GetPromClusterLabel())
	resChNodeCostPerRAMGiBHr := ctx.QueryAtTime(queryNodeCostPerRAMGiBHr, end)

	queryNodeCostPerGPUHr := fmt.Sprintf(queryFmtNodeCostPerGPUHr, durStr, env.GetPromClusterLabel())
	resChNodeCostPerGPUHr := ctx.QueryAtTime(queryNodeCostPerGPUHr, end)

	queryNodeIsSpot := fmt.Sprintf(queryFmtNodeIsSpot, durStr)
	resChNodeIsSpot := ctx.QueryAtTime(queryNodeIsSpot, end)

	queryPVCInfo := fmt.Sprintf(queryFmtPVCInfo, env.GetPromClusterLabel(), durStr, resStr)
	resChPVCInfo := ctx.QueryAtTime(queryPVCInfo, end)

	queryPodPVCAllocation := fmt.Sprintf(queryFmtPodPVCAllocation, durStr, env.GetPromClusterLabel())
	resChPodPVCAllocation := ctx.QueryAtTime(queryPodPVCAllocation, end)

	queryPVCBytesRequested := fmt.Sprintf(queryFmtPVCBytesRequested, durStr, env.GetPromClusterLabel())
	resChPVCBytesRequested := ctx.QueryAtTime(queryPVCBytesRequested, end)

	queryPVActiveMins := fmt.Sprintf(queryFmtPVActiveMins, env.GetPromClusterLabel(), durStr, resStr)
	resChPVActiveMins := ctx.QueryAtTime(queryPVActiveMins, end)

	queryPVBytes := fmt.Sprintf(queryFmtPVBytes, durStr, env.GetPromClusterLabel())
	resChPVBytes := ctx.QueryAtTime(queryPVBytes, end)

	queryPVCostPerGiBHour := fmt.Sprintf(queryFmtPVCostPerGiBHour, durStr, env.GetPromClusterLabel())
	resChPVCostPerGiBHour := ctx.QueryAtTime(queryPVCostPerGiBHour, end)

	queryNetTransferBytes := fmt.Sprintf(queryFmtNetTransferBytes, durStr, env.GetPromClusterLabel())
	resChNetTransferBytes := ctx.QueryAtTime(queryNetTransferBytes, end)

	queryNetReceiveBytes := fmt.Sprintf(queryFmtNetReceiveBytes, durStr, env.GetPromClusterLabel())
	resChNetReceiveBytes := ctx.QueryAtTime(queryNetReceiveBytes, end)

	queryNetZoneGiB := fmt.Sprintf(queryFmtNetZoneGiB, durStr, env.GetPromClusterLabel())
	resChNetZoneGiB := ctx.QueryAtTime(queryNetZoneGiB, end)

	queryNetZoneCostPerGiB := fmt.Sprintf(queryFmtNetZoneCostPerGiB, durStr, env.GetPromClusterLabel())
	resChNetZoneCostPerGiB := ctx.QueryAtTime(queryNetZoneCostPerGiB, end)

	queryNetRegionGiB := fmt.Sprintf(queryFmtNetRegionGiB, durStr, env.GetPromClusterLabel())
	resChNetRegionGiB := ctx.QueryAtTime(queryNetRegionGiB, end)

	queryNetRegionCostPerGiB := fmt.Sprintf(queryFmtNetRegionCostPerGiB, durStr, env.GetPromClusterLabel())
	resChNetRegionCostPerGiB := ctx.QueryAtTime(queryNetRegionCostPerGiB, end)

	queryNetInternetGiB := fmt.Sprintf(queryFmtNetInternetGiB, durStr, env.GetPromClusterLabel())
	resChNetInternetGiB := ctx.QueryAtTime(queryNetInternetGiB, end)

	queryNetInternetCostPerGiB := fmt.Sprintf(queryFmtNetInternetCostPerGiB, durStr, env.GetPromClusterLabel())
	resChNetInternetCostPerGiB := ctx.QueryAtTime(queryNetInternetCostPerGiB, end)

	queryNamespaceLabels := fmt.Sprintf(queryFmtNamespaceLabels, durStr)
	resChNamespaceLabels := ctx.QueryAtTime(queryNamespaceLabels, end)

	queryNamespaceAnnotations := fmt.Sprintf(queryFmtNamespaceAnnotations, durStr)
	resChNamespaceAnnotations := ctx.QueryAtTime(queryNamespaceAnnotations, end)

	queryPodLabels := fmt.Sprintf(queryFmtPodLabels, durStr)
	resChPodLabels := ctx.QueryAtTime(queryPodLabels, end)

	queryPodAnnotations := fmt.Sprintf(queryFmtPodAnnotations, durStr)
	resChPodAnnotations := ctx.QueryAtTime(queryPodAnnotations, end)

	queryServiceLabels := fmt.Sprintf(queryFmtServiceLabels, durStr)
	resChServiceLabels := ctx.QueryAtTime(queryServiceLabels, end)

	queryDeploymentLabels := fmt.Sprintf(queryFmtDeploymentLabels, durStr)
	resChDeploymentLabels := ctx.QueryAtTime(queryDeploymentLabels, end)

	queryStatefulSetLabels := fmt.Sprintf(queryFmtStatefulSetLabels, durStr)
	resChStatefulSetLabels := ctx.QueryAtTime(queryStatefulSetLabels, end)

	queryDaemonSetLabels := fmt.Sprintf(queryFmtDaemonSetLabels, durStr, env.GetPromClusterLabel())
	resChDaemonSetLabels := ctx.QueryAtTime(queryDaemonSetLabels, end)

	queryPodsWithReplicaSetOwner := fmt.Sprintf(queryFmtPodsWithReplicaSetOwner, durStr, env.GetPromClusterLabel())
	resChPodsWithReplicaSetOwner := ctx.QueryAtTime(queryPodsWithReplicaSetOwner, end)

	queryReplicaSetsWithoutOwners := fmt.Sprintf(queryFmtReplicaSetsWithoutOwners, durStr, env.GetPromClusterLabel())
	resChReplicaSetsWithoutOwners := ctx.QueryAtTime(queryReplicaSetsWithoutOwners, end)

	queryJobLabels := fmt.Sprintf(queryFmtJobLabels, durStr, env.GetPromClusterLabel())
	resChJobLabels := ctx.QueryAtTime(queryJobLabels, end)

	queryLBCostPerHr := fmt.Sprintf(queryFmtLBCostPerHr, durStr, env.GetPromClusterLabel())
	resChLBCostPerHr := ctx.QueryAtTime(queryLBCostPerHr, end)

	queryLBActiveMins := fmt.Sprintf(queryFmtLBActiveMins, env.GetPromClusterLabel(), durStr, resStr)
	resChLBActiveMins := ctx.QueryAtTime(queryLBActiveMins, end)

	resCPUCoresAllocated, _ := resChCPUCoresAllocated.Await()
	resCPURequests, _ := resChCPURequests.Await()
	resCPUUsageAvg, _ := resChCPUUsageAvg.Await()
	resCPUUsageMax, _ := resChCPUUsageMax.Await()
	resRAMBytesAllocated, _ := resChRAMBytesAllocated.Await()
	resRAMRequests, _ := resChRAMRequests.Await()
	resRAMUsageAvg, _ := resChRAMUsageAvg.Await()
	resRAMUsageMax, _ := resChRAMUsageMax.Await()
	resGPUsRequested, _ := resChGPUsRequested.Await()
	resGPUsAllocated, _ := resChGPUsAllocated.Await()

	resNodeCostPerCPUHr, _ := resChNodeCostPerCPUHr.Await()
	resNodeCostPerRAMGiBHr, _ := resChNodeCostPerRAMGiBHr.Await()
	resNodeCostPerGPUHr, _ := resChNodeCostPerGPUHr.Await()
	resNodeIsSpot, _ := resChNodeIsSpot.Await()

	resPVActiveMins, _ := resChPVActiveMins.Await()
	resPVBytes, _ := resChPVBytes.Await()
	resPVCostPerGiBHour, _ := resChPVCostPerGiBHour.Await()

	resPVCInfo, _ := resChPVCInfo.Await()
	resPVCBytesRequested, _ := resChPVCBytesRequested.Await()
	resPodPVCAllocation, _ := resChPodPVCAllocation.Await()

	resNetTransferBytes, _ := resChNetTransferBytes.Await()
	resNetReceiveBytes, _ := resChNetReceiveBytes.Await()
	resNetZoneGiB, _ := resChNetZoneGiB.Await()
	resNetZoneCostPerGiB, _ := resChNetZoneCostPerGiB.Await()
	resNetRegionGiB, _ := resChNetRegionGiB.Await()
	resNetRegionCostPerGiB, _ := resChNetRegionCostPerGiB.Await()
	resNetInternetGiB, _ := resChNetInternetGiB.Await()
	resNetInternetCostPerGiB, _ := resChNetInternetCostPerGiB.Await()

	resNamespaceLabels, _ := resChNamespaceLabels.Await()
	resNamespaceAnnotations, _ := resChNamespaceAnnotations.Await()
	resPodLabels, _ := resChPodLabels.Await()
	resPodAnnotations, _ := resChPodAnnotations.Await()
	resServiceLabels, _ := resChServiceLabels.Await()
	resDeploymentLabels, _ := resChDeploymentLabels.Await()
	resStatefulSetLabels, _ := resChStatefulSetLabels.Await()
	resDaemonSetLabels, _ := resChDaemonSetLabels.Await()
	resPodsWithReplicaSetOwner, _ := resChPodsWithReplicaSetOwner.Await()
	resReplicaSetsWithoutOwners, _ := resChReplicaSetsWithoutOwners.Await()
	resJobLabels, _ := resChJobLabels.Await()
	resLBCostPerHr, _ := resChLBCostPerHr.Await()
	resLBActiveMins, _ := resChLBActiveMins.Await()

	if ctx.HasErrors() {
		for _, err := range ctx.Errors() {
			log.Errorf("CostModel.ComputeAllocation: query context error %s", err)
		}

		return allocSet, ctx.ErrorCollection()
	}

	// We choose to apply allocation before requests in the cases of RAM and
	// CPU so that we can assert that allocation should always be greater than
	// or equal to request.
	applyCPUCoresAllocated(podMap, resCPUCoresAllocated, podUIDKeyMap)
	applyCPUCoresRequested(podMap, resCPURequests, podUIDKeyMap)
	applyCPUCoresUsedAvg(podMap, resCPUUsageAvg, podUIDKeyMap)
	applyCPUCoresUsedMax(podMap, resCPUUsageMax, podUIDKeyMap)
	applyRAMBytesAllocated(podMap, resRAMBytesAllocated, podUIDKeyMap)
	applyRAMBytesRequested(podMap, resRAMRequests, podUIDKeyMap)
	applyRAMBytesUsedAvg(podMap, resRAMUsageAvg, podUIDKeyMap)
	applyRAMBytesUsedMax(podMap, resRAMUsageMax, podUIDKeyMap)
	applyGPUsAllocated(podMap, resGPUsRequested, resGPUsAllocated, podUIDKeyMap)
	applyNetworkTotals(podMap, resNetTransferBytes, resNetReceiveBytes, podUIDKeyMap)
	applyNetworkAllocation(podMap, resNetZoneGiB, resNetZoneCostPerGiB, podUIDKeyMap)
	applyNetworkAllocation(podMap, resNetRegionGiB, resNetRegionCostPerGiB, podUIDKeyMap)
	applyNetworkAllocation(podMap, resNetInternetGiB, resNetInternetCostPerGiB, podUIDKeyMap)

	// In the case that a two pods with the same name had different containers,
	// we will double-count the containers. There is no way to associate each
	// container with the proper pod from the usage metrics above. This will
	// show up as a pod having two Allocations running for the whole pod runtime.

	// Other than that case, Allocations should be associated with pods by the
	// above functions.

	namespaceLabels := resToNamespaceLabels(resNamespaceLabels)
	podLabels := resToPodLabels(resPodLabels, podUIDKeyMap, ingestPodUID)
	namespaceAnnotations := resToNamespaceAnnotations(resNamespaceAnnotations)
	podAnnotations := resToPodAnnotations(resPodAnnotations, podUIDKeyMap, ingestPodUID)
	applyLabels(podMap, namespaceLabels, podLabels)
	applyAnnotations(podMap, namespaceAnnotations, podAnnotations)

	podDeploymentMap := labelsToPodControllerMap(podLabels, resToDeploymentLabels(resDeploymentLabels))
	podStatefulSetMap := labelsToPodControllerMap(podLabels, resToStatefulSetLabels(resStatefulSetLabels))
	podDaemonSetMap := resToPodDaemonSetMap(resDaemonSetLabels, podUIDKeyMap, ingestPodUID)
	podJobMap := resToPodJobMap(resJobLabels, podUIDKeyMap, ingestPodUID)
	podReplicaSetMap := resToPodReplicaSetMap(resPodsWithReplicaSetOwner, resReplicaSetsWithoutOwners, podUIDKeyMap, ingestPodUID)
	applyControllersToPods(podMap, podDeploymentMap)
	applyControllersToPods(podMap, podStatefulSetMap)
	applyControllersToPods(podMap, podDaemonSetMap)
	applyControllersToPods(podMap, podJobMap)
	applyControllersToPods(podMap, podReplicaSetMap)

	serviceLabels := getServiceLabels(resServiceLabels)
	allocsByService := map[serviceKey][]*kubecost.Allocation{}
	applyServicesToPods(podMap, podLabels, allocsByService, serviceLabels)

	// TODO breakdown network costs?

	// Build out the map of all PVs with class, size and cost-per-hour.
	// Note: this does not record time running, which we may want to
	// include later for increased PV precision. (As long as the PV has
	// a PVC, we get time running there, so this is only inaccurate
	// for short-lived, unmounted PVs.)
	pvMap := map[pvKey]*pv{}
	buildPVMap(resolution, pvMap, resPVCostPerGiBHour, resPVActiveMins)
	applyPVBytes(pvMap, resPVBytes)

	// Build out the map of all PVCs with time running, bytes requested,
	// and connect to the correct PV from pvMap. (If no PV exists, that
	// is noted, but does not result in any allocation/cost.)
	pvcMap := map[pvcKey]*pvc{}
	buildPVCMap(resolution, pvcMap, pvMap, resPVCInfo)
	applyPVCBytesRequested(pvcMap, resPVCBytesRequested)

	// Build out the relationships of pods to their PVCs. This step
	// populates the pvc.Count field so that pvc allocation can be
	// split appropriately among each pod's container allocation.
	podPVCMap := map[podKey][]*pvc{}
	buildPodPVCMap(podPVCMap, pvMap, pvcMap, podMap, resPodPVCAllocation, podUIDKeyMap, ingestPodUID)
	applyPVCsToPods(window, podMap, podPVCMap, pvcMap)

	// Identify PVCs without pods and add pv costs to the unmounted Allocation for the pvc's cluster
	applyUnmountedPVCs(window, podMap, pvcMap)

	// Identify PVs without PVCs and add PV costs to the unmounted Allocation for the PV's cluster
	applyUnmountedPVs(window, podMap, pvMap, pvcMap)

	lbMap := make(map[serviceKey]*lbCost)
	getLoadBalancerCosts(lbMap, resLBCostPerHr, resLBActiveMins, resolution)
	applyLoadBalancersToPods(window, podMap, lbMap, allocsByService)

	// Build out a map of Nodes with resource costs, discounts, and node types
	// for converting resource allocation data to cumulative costs.
	nodeMap := map[nodeKey]*nodePricing{}

	applyNodeCostPerCPUHr(nodeMap, resNodeCostPerCPUHr)
	applyNodeCostPerRAMGiBHr(nodeMap, resNodeCostPerRAMGiBHr)
	applyNodeCostPerGPUHr(nodeMap, resNodeCostPerGPUHr)
	applyNodeSpot(nodeMap, resNodeIsSpot)
	applyNodeDiscount(nodeMap, cm)
	cm.applyNodesToPod(podMap, nodeMap)

	// (3) Build out AllocationSet from Pod map
	for _, pod := range podMap {
		for _, alloc := range pod.Allocations {
			cluster := alloc.Properties.Cluster
			nodeName := alloc.Properties.Node
			namespace := alloc.Properties.Namespace
			podName := alloc.Properties.Pod
			container := alloc.Properties.Container

			// Make sure that the name is correct (node may not be present at this
			// point due to it missing from queryMinutes) then insert.
			alloc.Name = fmt.Sprintf("%s/%s/%s/%s/%s", cluster, nodeName, namespace, podName, container)
			allocSet.Set(alloc)
		}
	}

	return allocSet, nil
}
