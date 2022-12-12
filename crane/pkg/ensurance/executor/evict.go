package executor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	execsort "github.com/gocrane/crane/pkg/ensurance/executor/sort"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
)

type EvictExecutor struct {
	EvictPods EvictPods
	// All metrics(not only can be quantified metrics) metioned in triggerd NodeQOS and their corresponding watermarks
	EvictWatermark Watermarks
}

type EvictPods []podinfo.PodContext

func (e EvictPods) Find(key types.NamespacedName) int {
	for i, v := range e {
		if v.Key == key {
			return i
		}
	}

	return -1
}

func (e *EvictExecutor) Avoid(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentEvict), metrics.StepAvoid, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentEvict), metrics.StepAvoid, start)

	klog.V(6).Infof("EvictExecutor avoid, %v", *e)

	if len(e.EvictPods) == 0 {
		metrics.UpdateExecutorStatus(metrics.SubComponentEvict, metrics.StepAvoid, 0.0)
		return nil
	}

	metrics.UpdateExecutorStatus(metrics.SubComponentEvict, metrics.StepAvoid, 1.0)
	metrics.ExecutorStatusCounterInc(metrics.SubComponentEvict, metrics.StepAvoid)

	var errPodKeys, errKeys []string
	// TODO: totalReleasedResource used for prom metrics
	totalReleased := ReleaseResource{}

	/* The step to evict:
	1. If EvictWatermark has metrics that can't be quantified, select a evictable metric which has the highest action priority, use its EvictFunc to evict all selected pods, then return
	2. Get the gaps between current usage and watermarks
		2.1 If there is a metric that can't get current usage, select a evictable metric which has the highest action priority, use its EvictFunc to evict all selected pods, then return
		2.2 Traverse metrics that can be quantified, if there is gap for the metric, then sort candidate pods by its SortFunc if exists, otherwise use GeneralSorter by default.
	       Then evict sorted pods one by one util there is no gap to watermark
	*/

	quantified, notQuantified := e.EvictWatermark.DivideMetricsByEvictQuantified()

	// There is a metric that can't be EvictQuantified, so evict all selected pods
	if len(notQuantified) != 0 {
		klog.V(6).Info("There is a metric that can't be EvcitQuantified")

		highestPriorityMetric := e.EvictWatermark.GetHighestPriorityEvictableMetric()
		if highestPriorityMetric != "" {
			klog.V(6).Infof("The highestPriorityMetric is %s", highestPriorityMetric)
			errPodKeys = e.evictPods(ctx, &totalReleased, highestPriorityMetric)
		}
	} else {
		ctx.ToBeEvict = calculateGaps(ctx.stateMap, nil, e, ctx.executeExcessPercent)

		if ctx.ToBeEvict.HasUsageMissedMetric() {
			klog.V(6).Infof("There is a metric usage missed")
			highestPriorityMetric := e.EvictWatermark.GetHighestPriorityEvictableMetric()
			if highestPriorityMetric != "" {
				errPodKeys = e.evictPods(ctx, &totalReleased, highestPriorityMetric)
			}
		} else {
			// The metrics in ToBeEvict are can be EvictQuantified and has current usage, then evict precisely
			var released ReleaseResource
			wg := sync.WaitGroup{}
			for _, m := range quantified {
				klog.V(6).Infof("Evict precisely on metric %s, and current gaps are %+v", m, ctx.ToBeEvict)
				if metricMap[m].Sortable {
					metricMap[m].SortFunc(e.EvictPods)
				} else {
					execsort.GeneralSorter(e.EvictPods)
				}

				for _, pc := range e.EvictPods {
					klog.V(6).Info(pc.Key.String())
				}
				for !ctx.ToBeEvict.TargetGapsRemoved(m) {
					klog.V(2).Infof("For metric %s, there is gap %f to watermarks %s", m, ctx.ToBeEvict[m], m)
					if podinfo.ContainsNoExecutedPod(e.EvictPods) {
						index := podinfo.GetFirstPendingPod(e.EvictPods)
						errKeys, released = metricMap[m].EvictFunc(&wg, ctx, index, &totalReleased, e.EvictPods)
						errPodKeys = append(errPodKeys, errKeys...)
						klog.Warningf("Evicted pods %s, released %f of %s", e.EvictPods[index].Key, released[m], m)
						e.EvictPods[index].Executed = true
						ctx.ToBeEvict[m] -= released[m]
					} else {
						klog.V(6).Info("There is no pod that can be evicted")
						break
					}
				}
			}
			wg.Wait()
		}
	}

	if len(errPodKeys) != 0 {
		return fmt.Errorf("some pod evict failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (e *EvictExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}

func (e *EvictExecutor) evictPods(ctx *ExecuteContext, totalReleasedResource *ReleaseResource, m WatermarkMetric) (errPodKeys []string) {
	wg := sync.WaitGroup{}
	for i := range e.EvictPods {
		errKeys, _ := metricMap[m].EvictFunc(&wg, ctx, i, totalReleasedResource, e.EvictPods)
		errPodKeys = append(errPodKeys, errKeys...)
	}
	wg.Wait()
	return
}
