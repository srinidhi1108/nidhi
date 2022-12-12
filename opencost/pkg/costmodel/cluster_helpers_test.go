package costmodel

import (
	"reflect"
	"testing"
	"time"

	"github.com/opencost/opencost/pkg/cloud"
	"github.com/opencost/opencost/pkg/config"
	"github.com/opencost/opencost/pkg/prom"
	"github.com/opencost/opencost/pkg/util"

	"github.com/davecgh/go-spew/spew"
)

func TestMergeTypeMaps(t *testing.T) {
	cases := []struct {
		name     string
		map1     map[nodeIdentifierNoProviderID]string
		map2     map[nodeIdentifierNoProviderID]string
		expected map[nodeIdentifierNoProviderID]string
	}{
		{
			name:     "both empty",
			map1:     map[nodeIdentifierNoProviderID]string{},
			map2:     map[nodeIdentifierNoProviderID]string{},
			expected: map[nodeIdentifierNoProviderID]string{},
		},
		{
			name: "map2 empty",
			map1: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			map2: map[nodeIdentifierNoProviderID]string{},
			expected: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
		},
		{
			name: "map1 empty",
			map1: map[nodeIdentifierNoProviderID]string{},
			map2: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			expected: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
		},
		{
			name: "no overlap",
			map1: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			map2: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: "type2",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node4",
				}: "type4",
			},
			expected: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: "type2",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node4",
				}: "type4",
			},
		},
		{
			name: "with overlap",
			map1: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			map2: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: "type2",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type4",
			},
			expected: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: "type2",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := mergeTypeMaps(testCase.map1, testCase.map2)

			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("mergeTypeMaps case %s failed. Got %+v but expected %+v", testCase.name, result, testCase.expected)
			}
		})
	}
}

func TestBuildNodeMap(t *testing.T) {
	cases := []struct {
		name                 string
		cpuCostMap           map[NodeIdentifier]float64
		ramCostMap           map[NodeIdentifier]float64
		gpuCostMap           map[NodeIdentifier]float64
		gpuCountMap          map[NodeIdentifier]float64
		cpuCoresMap          map[nodeIdentifierNoProviderID]float64
		ramBytesMap          map[nodeIdentifierNoProviderID]float64
		ramUserPctMap        map[nodeIdentifierNoProviderID]float64
		ramSystemPctMap      map[nodeIdentifierNoProviderID]float64
		cpuBreakdownMap      map[nodeIdentifierNoProviderID]*ClusterCostsBreakdown
		activeDataMap        map[NodeIdentifier]activeData
		preemptibleMap       map[NodeIdentifier]bool
		labelsMap            map[nodeIdentifierNoProviderID]map[string]string
		clusterAndNameToType map[nodeIdentifierNoProviderID]string
		expected             map[NodeIdentifier]*Node
	}{
		{
			name:     "empty",
			expected: map[NodeIdentifier]*Node{},
		},
		{
			name: "just cpu cost",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: 0.048,
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1",
					NodeType:     "type1",
					CPUCost:      0.048,
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
		{
			name: "just cpu cost with empty provider ID",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster: "cluster1",
					Name:    "node1",
				}: 0.048,
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster: "cluster1",
					Name:    "node1",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					NodeType:     "type1",
					CPUCost:      0.048,
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
		{
			name: "cpu cost with overlapping node names",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: 0.048,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: 0.087,
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1_A",
					NodeType:     "type1",
					CPUCost:      0.048,
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1_B",
					NodeType:     "type1",
					CPUCost:      0.087,
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
		{
			name: "all fields + overlapping node names",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: 0.048,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: 0.087,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: 0.033,
			},
			ramCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: 0.09,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: 0.3,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: 0.024,
			},
			gpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: 0.8,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: 1.4,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: 3.1,
			},
			gpuCountMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: 1.0,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: 1.0,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: 2.0,
			},
			cpuCoresMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 2.0,
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: 5.0,
			},
			ramBytesMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 2048.0,
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: 6303.0,
			},
			ramUserPctMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 30.0,
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: 42.6,
			},
			ramSystemPctMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 15.0,
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: 20.1,
			},
			cpuBreakdownMap: map[nodeIdentifierNoProviderID]*ClusterCostsBreakdown{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: &ClusterCostsBreakdown{
					System: 20.2,
					User:   68.0,
				},
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: &ClusterCostsBreakdown{
					System: 28.9,
					User:   34.0,
				},
			},
			activeDataMap: map[NodeIdentifier]activeData{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: activeData{
					start:   time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					end:     time.Date(2020, 6, 16, 9, 20, 39, 0, time.UTC),
					minutes: 5*60 + 35 + (11.0 / 60.0),
				},
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: activeData{
					start:   time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					end:     time.Date(2020, 6, 16, 9, 21, 39, 0, time.UTC),
					minutes: 5*60 + 36 + (11.0 / 60.0),
				},
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: activeData{
					start:   time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					end:     time.Date(2020, 6, 16, 9, 10, 39, 0, time.UTC),
					minutes: 5*60 + 25 + (11.0 / 60.0),
				},
			},
			preemptibleMap: map[NodeIdentifier]bool{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: true,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: false,
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: false,
			},
			labelsMap: map[nodeIdentifierNoProviderID]map[string]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: map[string]string{
					"labelname1_A": "labelvalue1_A",
					"labelname1_B": "labelvalue1_B",
				},
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: map[string]string{
					"labelname2_A": "labelvalue2_A",
					"labelname2_B": "labelvalue2_B",
				},
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "type1",
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node2",
				}: "type2",
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
				}: &Node{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_A",
					NodeType:   "type1",
					CPUCost:    0.048,
					RAMCost:    0.09,
					GPUCost:    0.8,
					CPUCores:   2.0,
					GPUCount:   1.0,
					RAMBytes:   2048.0,
					RAMBreakdown: &ClusterCostsBreakdown{
						User:   30.0,
						System: 15.0,
					},
					CPUBreakdown: &ClusterCostsBreakdown{
						System: 20.2,
						User:   68.0,
					},
					Start:       time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					End:         time.Date(2020, 6, 16, 9, 20, 39, 0, time.UTC),
					Minutes:     5*60 + 35 + (11.0 / 60.0),
					Preemptible: true,
					Labels: map[string]string{
						"labelname1_A": "labelvalue1_A",
						"labelname1_B": "labelvalue1_B",
					},
				},
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
				}: &Node{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1_B",
					NodeType:   "type1",
					CPUCost:    0.087,
					RAMCost:    0.3,
					GPUCost:    1.4,
					CPUCores:   2.0,
					GPUCount:   1.0,
					RAMBytes:   2048.0,
					RAMBreakdown: &ClusterCostsBreakdown{
						User:   30.0,
						System: 15.0,
					},
					CPUBreakdown: &ClusterCostsBreakdown{
						System: 20.2,
						User:   68.0,
					},
					Start:       time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					End:         time.Date(2020, 6, 16, 9, 21, 39, 0, time.UTC),
					Minutes:     5*60 + 36 + (11.0 / 60.0),
					Preemptible: false,
					Labels: map[string]string{
						"labelname1_A": "labelvalue1_A",
						"labelname1_B": "labelvalue1_B",
					},
				},
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
				}: &Node{
					Cluster:    "cluster1",
					Name:       "node2",
					ProviderID: "prov_node2_A",
					NodeType:   "type2",
					CPUCost:    0.033,
					RAMCost:    0.024,
					GPUCost:    3.1,
					CPUCores:   5.0,
					GPUCount:   2.0,
					RAMBytes:   6303.0,
					RAMBreakdown: &ClusterCostsBreakdown{
						User:   42.6,
						System: 20.1,
					},
					CPUBreakdown: &ClusterCostsBreakdown{
						System: 28.9,
						User:   34.0,
					},
					Start:       time.Date(2020, 6, 16, 3, 45, 28, 0, time.UTC),
					End:         time.Date(2020, 6, 16, 9, 10, 39, 0, time.UTC),
					Minutes:     5*60 + 25 + (11.0 / 60.0),
					Preemptible: false,
					Labels: map[string]string{
						"labelname2_A": "labelvalue2_A",
						"labelname2_B": "labelvalue2_B",
					},
				},
			},
		},
		{
			name: "e2-micro cpu cost adjustment",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: 0.048,
			},
			cpuCoresMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 6.0, // GKE lies about number of cores
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "e2-micro", // for this node type
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1",
					NodeType:     "e2-micro",
					CPUCost:      0.048 * (partialCPUMap["e2-micro"] / 6.0), // adjustmentFactor is (v / GKE cores)
					CPUCores:     partialCPUMap["e2-micro"],
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
		{
			name: "e2-small cpu cost adjustment",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: 0.048,
			},
			cpuCoresMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 6.0, // GKE lies about number of cores
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "e2-small", // for this node type
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1",
					NodeType:     "e2-small",
					CPUCost:      0.048 * (partialCPUMap["e2-small"] / 6.0), // adjustmentFactor is (v / GKE cores)
					CPUCores:     partialCPUMap["e2-small"],
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
		{
			name: "e2-medium cpu cost adjustment",
			cpuCostMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: 0.048,
			},
			cpuCoresMap: map[nodeIdentifierNoProviderID]float64{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: 6.0, // GKE lies about number of cores
			},
			clusterAndNameToType: map[nodeIdentifierNoProviderID]string{
				nodeIdentifierNoProviderID{
					Cluster: "cluster1",
					Name:    "node1",
				}: "e2-medium", // for this node type
			},
			expected: map[NodeIdentifier]*Node{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "prov_node1",
				}: &Node{
					Cluster:      "cluster1",
					Name:         "node1",
					ProviderID:   "prov_node1",
					NodeType:     "e2-medium",
					CPUCost:      0.048 * (partialCPUMap["e2-medium"] / 6.0), // adjustmentFactor is (v / GKE cores)
					CPUCores:     partialCPUMap["e2-medium"],
					CPUBreakdown: &ClusterCostsBreakdown{},
					RAMBreakdown: &ClusterCostsBreakdown{},
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildNodeMap(
				testCase.cpuCostMap, testCase.ramCostMap, testCase.gpuCostMap, testCase.gpuCountMap,
				testCase.cpuCoresMap, testCase.ramBytesMap, testCase.ramUserPctMap,
				testCase.ramSystemPctMap,
				testCase.cpuBreakdownMap,
				testCase.activeDataMap,
				testCase.preemptibleMap,
				testCase.labelsMap,
				testCase.clusterAndNameToType,
				time.Minute,
			)

			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("buildNodeMap case %s failed. Got %+v but expected %+v", testCase.name, result, testCase.expected)

				// Use spew because we have to follow pointers to figure out
				// what isn't matching up
				t.Logf("Got: %s", spew.Sdump(result))
				t.Logf("Expected: %s", spew.Sdump(testCase.expected))
			}
		})
	}
}

func TestBuildGPUCostMap(t *testing.T) {
	cases := []struct {
		name       string
		promResult []*prom.QueryResult
		countMap   map[NodeIdentifier]float64
		expected   map[NodeIdentifier]float64
	}{
		{
			name: "All Zeros",
			promResult: []*prom.QueryResult{
				{
					Metric: map[string]interface{}{
						"cluster_id":    "cluster1",
						"node":          "node1",
						"instance_type": "type1",
						"provider_id":   "provider1",
					},
					Values: []*util.Vector{
						&util.Vector{
							Timestamp: 0,
							Value:     0,
						},
					},
				},
			},
			countMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
			expected: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
		},
		{
			name: "Zero Node Count",
			promResult: []*prom.QueryResult{
				{
					Metric: map[string]interface{}{
						"cluster_id":    "cluster1",
						"node":          "node1",
						"instance_type": "type1",
						"provider_id":   "provider1",
					},
					Values: []*util.Vector{
						&util.Vector{
							Timestamp: 0,
							Value:     2,
						},
					},
				},
			},
			countMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
			expected: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
		},
		{
			name: "Missing Node Count",
			promResult: []*prom.QueryResult{
				{
					Metric: map[string]interface{}{
						"cluster_id":    "cluster1",
						"node":          "node1",
						"instance_type": "type1",
						"provider_id":   "provider1",
					},
					Values: []*util.Vector{
						&util.Vector{
							Timestamp: 0,
							Value:     2,
						},
					},
				},
			},
			countMap: map[NodeIdentifier]float64{},
			expected: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
		},
		{
			name: "missing cost data",
			promResult: []*prom.QueryResult{
				{},
			},
			countMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 0,
			},
			expected: map[NodeIdentifier]float64{},
		},
		{
			name: "All values present",
			promResult: []*prom.QueryResult{
				{
					Metric: map[string]interface{}{
						"cluster_id":    "cluster1",
						"node":          "node1",
						"instance_type": "type1",
						"provider_id":   "provider1",
					},
					Values: []*util.Vector{
						&util.Vector{
							Timestamp: 0,
							Value:     2,
						},
					},
				},
			},
			countMap: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 2,
			},
			expected: map[NodeIdentifier]float64{
				NodeIdentifier{
					Cluster:    "cluster1",
					Name:       "node1",
					ProviderID: "provider1",
				}: 4,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testProvider := &cloud.CustomProvider{
				Config: cloud.NewProviderConfig(config.NewConfigFileManager(nil), "fakeFile"),
			}
			testPreemptible := make(map[NodeIdentifier]bool)
			result, _ := buildGPUCostMap(testCase.promResult, testCase.countMap, testProvider, testPreemptible)
			if !reflect.DeepEqual(result, testCase.expected) {
				t.Errorf("buildGPUCostMap case %s failed. Got %+v but expected %+v", testCase.name, result, testCase.expected)
			}
		})
	}
}

func TestAssetCustompricing(t *testing.T) {

	nodePromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":    "cluster1",
				"node":          "node1",
				"instance_type": "type1",
				"provider_id":   "provider1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     0.5,
				},
			},
		},
	}

	pvCostPromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":       "cluster1",
				"persistentvolume": "pvc1",
				"provider_id":      "provider1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1.0,
				},
			},
		},
	}

	pvSizePromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":       "cluster1",
				"persistentvolume": "pvc1",
				"provider_id":      "provider1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1073741824.0,
				},
			},
		},
	}

	pvMinsPromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":       "cluster1",
				"persistentvolume": "pvc1",
				"provider_id":      "provider1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1.0,
				},
				&util.Vector{
					Timestamp: 3600.0,
					Value:     1.0,
				},
			},
		},
	}

	pvAvgUsagePromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":            "cluster1",
				"persistentvolumeclaim": "pv-claim1",
				"namespace":             "ns1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1.0,
				},
				&util.Vector{
					Timestamp: 3600.0,
					Value:     1.0,
				},
			},
		},
	}

	pvMaxUsagePromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":            "cluster1",
				"persistentvolumeclaim": "pv-claim1",
				"namespace":             "ns1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1.0,
				},
				&util.Vector{
					Timestamp: 3600.0,
					Value:     1.0,
				},
			},
		},
	}

	pvInfoPromResult := []*prom.QueryResult{
		{
			Metric: map[string]interface{}{
				"cluster_id":            "cluster1",
				"persistentvolumeclaim": "pv-claim1",
				"volumename":            "pvc1",
				"namespace":             "ns1",
			},
			Values: []*util.Vector{
				&util.Vector{
					Timestamp: 0,
					Value:     1.0,
				},
			},
		},
	}

	gpuCountMap := map[NodeIdentifier]float64{
		NodeIdentifier{
			Cluster:    "cluster1",
			Name:       "node1",
			ProviderID: "provider1",
		}: 2,
	}

	nodeKey := NodeIdentifier{
		Cluster:    "cluster1",
		Name:       "node1",
		ProviderID: "provider1",
	}

	cases := []struct {
		name             string
		customPricingMap map[string]string
		expectedPricing  map[string]float64
	}{
		{
			name:             "No custom pricing",
			customPricingMap: map[string]string{},
			expectedPricing: map[string]float64{
				"CPU":     0.5,
				"RAM":     0.5,
				"GPU":     1.0,
				"Storage": 1.0,
			},
		},
		{
			name: "Custom pricing enabled",
			customPricingMap: map[string]string{
				"CPU":                 "20.0",
				"RAM":                 "4.0",
				"GPU":                 "500.0",
				"Storage":             "0.1",
				"customPricesEnabled": "true",
			},
			expectedPricing: map[string]float64{
				"CPU":     0.027397,              // 20.0 / 730
				"RAM":     5.102716386318207e-12, // 4.0 / 730 / 1024^3
				"GPU":     1.369864,              // 500.0 / 730 * 2
				"Storage": 0.000137,              // 0.1 / 730 * (1073741824.0 / 1024 / 1024 / 1024) * (60 / 60) => 0.1 / 730 * 1 * 1
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testProvider := &cloud.CustomProvider{
				Config: cloud.NewProviderConfig(config.NewConfigFileManager(nil), ""),
			}
			testProvider.UpdateConfigFromConfigMap(testCase.customPricingMap)

			testPreemptible := make(map[NodeIdentifier]bool)
			cpuMap, _ := buildCPUCostMap(nodePromResult, testProvider, testPreemptible)
			ramMap, _ := buildRAMCostMap(nodePromResult, testProvider, testPreemptible)
			gpuMap, _ := buildGPUCostMap(nodePromResult, gpuCountMap, testProvider, testPreemptible)

			cpuResult := cpuMap[nodeKey]
			ramResult := ramMap[nodeKey]
			gpuResult := gpuMap[nodeKey]

			diskMap := map[DiskIdentifier]*Disk{}
			pvCosts(diskMap, time.Hour, pvMinsPromResult, pvSizePromResult, pvCostPromResult, pvAvgUsagePromResult, pvMaxUsagePromResult, pvInfoPromResult, testProvider)

			diskResult := diskMap[DiskIdentifier{"cluster1", "pvc1"}].Cost

			if !util.IsApproximately(cpuResult, testCase.expectedPricing["CPU"]) {
				t.Errorf("CPU custom pricing error in %s. Got %v but expected %v", testCase.name, cpuResult, testCase.expectedPricing["CPU"])
			}
			if !util.IsApproximately(ramResult, testCase.expectedPricing["RAM"]) {
				t.Errorf("RAM custom pricing error in %s. Got %v but expected %v", testCase.name, ramResult, testCase.expectedPricing["RAM"])
			}
			if !util.IsApproximately(gpuResult, testCase.expectedPricing["GPU"]) {
				t.Errorf("GPU custom pricing error in %s. Got %v but expected %v", testCase.name, gpuResult, testCase.expectedPricing["GPU"])
			}
			if !util.IsApproximately(diskResult, testCase.expectedPricing["Storage"]) {
				t.Errorf("Disk custom pricing error in %s. Got %v but expected %v", testCase.name, diskResult, testCase.expectedPricing["Storage"])
			}
		})
	}

}
