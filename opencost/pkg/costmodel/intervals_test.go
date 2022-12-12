package costmodel

import (
	"reflect"
	"testing"
	"time"

	"github.com/opencost/opencost/pkg/kubecost"
)

func TestGetIntervalPointsFromWindows(t *testing.T) {
	cases := []struct {
		name           string
		pvcIntervalMap map[podKey]kubecost.Window
		expected       []IntervalPoint
	}{
		{
			name: "four pods w/ various overlaps",
			pvcIntervalMap: map[podKey]kubecost.Window{
				// Pod running from 8 am to 9 am
				podKey{
					Pod: "Pod1",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
				// Pod running from 8:30 am to 9 am
				podKey{
					Pod: "Pod2",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
				// Pod running from 8:45 am to 9 am
				podKey{
					Pod: "Pod3",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 45, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
				// Pod running from 8 am to 8:15 am
				podKey{
					Pod: "Pod4",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 8, 15, 0, 0, time.UTC),
				)),
			},
			expected: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", podKey{Pod: "Pod4"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 15, 0, 0, time.UTC), "end", podKey{Pod: "Pod4"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", podKey{Pod: "Pod2"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 45, 0, 0, time.UTC), "start", podKey{Pod: "Pod3"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod2"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod3"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod1"}),
			},
		},
		{
			name: "two pods no overlap",
			pvcIntervalMap: map[podKey]kubecost.Window{
				// Pod running from 8 am to 8:30 am
				podKey{
					Pod: "Pod1",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC),
				)),
				// Pod running from 8:30 am to 9 am
				podKey{
					Pod: "Pod2",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
			},
			expected: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", podKey{Pod: "Pod2"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "end", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod2"}),
			},
		},
		{
			name: "two pods total overlap",
			pvcIntervalMap: map[podKey]kubecost.Window{
				// Pod running from 8:30 am to 9 am
				podKey{
					Pod: "Pod1",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
				// Pod running from 8:30 am to 9 am
				podKey{
					Pod: "Pod2",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
			},
			expected: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", podKey{Pod: "Pod2"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod2"}),
			},
		},
		{
			name: "one pod",
			pvcIntervalMap: map[podKey]kubecost.Window{
				// Pod running from 8 am to 9 am
				podKey{
					Pod: "Pod1",
				}: kubecost.Window(kubecost.NewClosedWindow(
					time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC),
					time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
				)),
			},
			expected: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", podKey{Pod: "Pod1"}),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", podKey{Pod: "Pod1"}),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {

			result := getIntervalPointsFromWindows(testCase.pvcIntervalMap)

			if len(result) != len(testCase.expected) {
				t.Errorf("getIntervalPointsFromWindows test failed: %s: Got %+v but expected %+v", testCase.name, result, testCase.expected)
			}

			for i := range testCase.expected {

				// For correctness in terms of individual position of IntervalPoints, we only need to check the time/type.
				// Key is used in other associated calculations, so it must exist, but order does not matter if other sorting
				// logic is obeyed.
				if !testCase.expected[i].Time.Equal(result[i].Time) || testCase.expected[i].PointType != result[i].PointType {
					t.Errorf("getIntervalPointsFromWindows test failed: %s: Got point %s:%s but expected %s:%s", testCase.name, testCase.expected[i].PointType, testCase.expected[i].Time, result[i].PointType, result[i].Time)
				}

			}

		})
	}
}

func TestGetPVCCostCoefficients(t *testing.T) {
	pvc1 := &pvc{
		Bytes:     0,
		Name:      "pvc1",
		Cluster:   "cluster1",
		Namespace: "namespace1",
		Start:     time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC),
		End:       time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC),
	}
	pod1Key := newPodKey("cluster1", "namespace1", "pod1")
	pod2Key := newPodKey("cluster1", "namespace1", "pod2")
	pod3Key := newPodKey("cluster1", "namespace1", "pod3")
	pod4Key := newPodKey("cluster1", "namespace1", "pod4")
	ummountedPodKey := newPodKey("cluster1", kubecost.UnmountedSuffix, kubecost.UnmountedSuffix)

	cases := []struct {
		name           string
		pvc            *pvc
		pvcIntervalMap map[podKey]kubecost.Window
		intervals      []IntervalPoint
		expected       map[podKey][]CoefficientComponent
	}{
		{
			name: "four pods w/ various overlaps",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", pod4Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 15, 0, 0, time.UTC), "end", pod4Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", pod2Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 45, 0, 0, time.UTC), "start", pod3Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod2Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod3Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod1Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{0.5, 0.25},
					CoefficientComponent{1, 0.25},
					CoefficientComponent{0.5, 0.25},
					CoefficientComponent{1.0 / 3.0, 0.25},
				},
				pod2Key: []CoefficientComponent{
					CoefficientComponent{0.5, 0.25},
					CoefficientComponent{1.0 / 3.0, 0.25},
				},
				pod3Key: []CoefficientComponent{
					CoefficientComponent{1.0 / 3.0, 0.25},
				},
				pod4Key: []CoefficientComponent{
					CoefficientComponent{0.5, 0.25},
				},
			},
		},
		{
			name: "two pods no overlap",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", pod2Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "end", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod2Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{1.0, 0.5},
				},
				pod2Key: []CoefficientComponent{
					CoefficientComponent{1.0, 0.5},
				},
			},
		},
		{
			name: "two pods total overlap",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 30, 0, 0, time.UTC), "start", pod2Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod2Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{0.5, 0.5},
				},
				pod2Key: []CoefficientComponent{
					CoefficientComponent{0.5, 0.5},
				},
				ummountedPodKey: []CoefficientComponent{
					CoefficientComponent{1.0, 0.5},
				},
			},
		},
		{
			name: "one pod",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod1Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{1.0, 1.0},
				},
			},
		},
		{
			name: "two pods with gap",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 0, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 15, 0, 0, time.UTC), "end", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 45, 0, 0, time.UTC), "start", pod2Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 9, 0, 0, 0, time.UTC), "end", pod2Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{1.0, 0.25},
				},
				pod2Key: []CoefficientComponent{
					CoefficientComponent{1.0, 0.25},
				},
				ummountedPodKey: []CoefficientComponent{
					CoefficientComponent{1.0, 0.5},
				},
			},
		},
		{
			name: "one pods start and end in window",
			pvc:  pvc1,
			intervals: []IntervalPoint{
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 15, 0, 0, time.UTC), "start", pod1Key),
				NewIntervalPoint(time.Date(2021, 2, 19, 8, 45, 0, 0, time.UTC), "end", pod1Key),
			},
			expected: map[podKey][]CoefficientComponent{
				pod1Key: []CoefficientComponent{
					CoefficientComponent{1.0, 0.5},
				},
				ummountedPodKey: []CoefficientComponent{
					CoefficientComponent{1.0, 0.25},
					CoefficientComponent{1.0, 0.25},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := getPVCCostCoefficients(testCase.intervals, testCase.pvc)

			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("getPVCCostCoefficients test failed: %s: Got %+v but expected %+v", testCase.name, result, testCase.expected)
			}

			// check that coefficients sum to 1, to ensure that 100% of PVC cost is being distributed
			sum := 0.0
			for _, coefs := range result {
				sum += getCoefficientFromComponents(coefs)
			}
			if sum != 1.0 {
				t.Errorf("getPVCCostCoefficients test failed: coefficient totals did not sum to 1.0: %f", sum)
			}
		})
	}
}
