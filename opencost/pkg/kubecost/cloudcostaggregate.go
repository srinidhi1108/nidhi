package kubecost

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/opencost/opencost/pkg/filter"
	"github.com/opencost/opencost/pkg/log"
)

const (
	CloudCostAccountProp  string = "account"
	CloudCostProjectProp  string = "project"
	CloudCostProviderProp string = "provider"
	CloudCostServiceProp  string = "service"
	CloudCostLabelProp    string = "label"
)

// CloudCostAggregateProperties unique property set for CloudCostAggregate within a window
type CloudCostAggregateProperties struct {
	Provider   string `json:"provider"`
	Account    string `json:"account"`
	Project    string `json:"project"`
	Service    string `json:"service"`
	LabelValue string `json:"label"`
}

func (ccap CloudCostAggregateProperties) Equal(that CloudCostAggregateProperties) bool {
	return ccap.Provider == that.Provider &&
		ccap.Account == that.Account &&
		ccap.Project == that.Project &&
		ccap.Service == that.Service &&
		ccap.LabelValue == that.LabelValue
}

func (ccap CloudCostAggregateProperties) Key(props []string) string {
	if len(props) == 0 {
		return fmt.Sprintf("%s/%s/%s/%s/%s", ccap.Provider, ccap.Account, ccap.Project, ccap.Service, ccap.LabelValue)
	}

	keys := make([]string, len(props))
	for i, prop := range props {
		key := UnallocatedSuffix

		switch prop {
		case CloudCostProviderProp:
			if ccap.Provider != "" {
				key = ccap.Provider
			}
		case CloudCostAccountProp:
			if ccap.Account != "" {
				key = ccap.Account
			}
		case CloudCostProjectProp:
			if ccap.Project != "" {
				key = ccap.Project
			}
		case CloudCostServiceProp:
			if ccap.Service != "" {
				key = ccap.Service
			}
		case CloudCostLabelProp:
			if ccap.LabelValue != "" {
				key = ccap.LabelValue
			}
		}

		keys[i] = key
	}

	return strings.Join(keys, "/")
}

// CloudCostAggregate represents an aggregation of Billing Integration data on the properties listed
// - KubernetesPercent is the percent of the CloudCostAggregates cost which was from an item which could be identified
//   as coming from a kubernetes resources.
// - Cost is the sum of the cost of each item in the CloudCostAggregate
// - Credit is the sum of credits applied to each item in the CloudCostAggregate

type CloudCostAggregate struct {
	Properties        CloudCostAggregateProperties `json:"properties"`
	KubernetesPercent float64                      `json:"kubernetesPercent"`
	Cost              float64                      `json:"cost"`
	Credit            float64                      `json:"credit"`
}

func (cca *CloudCostAggregate) Clone() *CloudCostAggregate {
	return &CloudCostAggregate{
		Properties:        cca.Properties,
		KubernetesPercent: cca.KubernetesPercent,
		Cost:              cca.Cost,
		Credit:            cca.Credit,
	}
}

func (cca *CloudCostAggregate) Equal(that *CloudCostAggregate) bool {
	if that == nil {
		return false
	}

	return cca.Cost == that.Cost &&
		cca.Credit == that.Credit &&
		cca.Properties.Equal(that.Properties)
}

func (cca *CloudCostAggregate) Key(props []string) string {
	return cca.Properties.Key(props)
}

func (cca *CloudCostAggregate) StringProperty(prop string) (string, error) {
	if cca == nil {
		return "", nil
	}

	switch prop {
	case CloudCostAccountProp:
		return cca.Properties.Account, nil
	case CloudCostProjectProp:
		return cca.Properties.Project, nil
	case CloudCostProviderProp:
		return cca.Properties.Provider, nil
	case CloudCostServiceProp:
		return cca.Properties.Service, nil
	case CloudCostLabelProp:
		return cca.Properties.LabelValue, nil
	default:
		return "", fmt.Errorf("invalid property name: %s", prop)
	}
}

func (cca *CloudCostAggregate) add(that *CloudCostAggregate) {
	if cca == nil {
		log.Warnf("cannot add to nil CloudCostAggregate")
		return
	}

	// Compute KubernetesPercent for sum
	k8sPct := 0.0
	sumCost := cca.Cost + that.Cost
	if sumCost > 0.0 {
		thisK8sCost := cca.Cost * cca.KubernetesPercent
		thatK8sCost := that.Cost * that.KubernetesPercent
		k8sPct = (thisK8sCost + thatK8sCost) / sumCost
	}

	cca.Cost = sumCost
	cca.Credit += that.Credit
	cca.KubernetesPercent = k8sPct
}

type CloudCostAggregateSet struct {
	CloudCostAggregates   map[string]*CloudCostAggregate `json:"items"`
	AggregationProperties []string                       `json:"-"`
	Integration           string                         `json:"-"`
	LabelName             string                         `json:"labelName,omitempty"`
	Window                Window                         `json:"window"`
}

func NewCloudCostAggregateSet(start, end time.Time, cloudCostAggregates ...*CloudCostAggregate) *CloudCostAggregateSet {
	ccas := &CloudCostAggregateSet{
		CloudCostAggregates: map[string]*CloudCostAggregate{},
		Window:              NewWindow(&start, &end),
	}

	for _, cca := range cloudCostAggregates {
		ccas.insertByProperty(cca, nil)
	}

	return ccas
}

func (ccas *CloudCostAggregateSet) Aggregate(props []string) (*CloudCostAggregateSet, error) {
	if ccas == nil {
		return nil, errors.New("cannot aggregate a nil CloudCostAggregateSet")
	}

	if ccas.Window.IsOpen() {
		return nil, fmt.Errorf("cannot aggregate a CloudCostAggregateSet with an open window: %s", ccas.Window)
	}

	// Create a new result set, with the given aggregation property
	result := NewCloudCostAggregateSet(*ccas.Window.Start(), *ccas.Window.End())
	result.AggregationProperties = props
	result.LabelName = ccas.LabelName
	result.Integration = ccas.Integration

	// Insert clones of each item in the set, keyed by the given property.
	// The underlying insert logic will add binned items together.
	for name, cca := range ccas.CloudCostAggregates {
		ccaClone := cca.Clone()
		err := result.insertByProperty(ccaClone, props)
		if err != nil {
			return nil, fmt.Errorf("error aggregating %s by %v: %s", name, props, err)
		}
	}

	return result, nil
}

func (ccas *CloudCostAggregateSet) Filter(filters filter.Filter[*CloudCostAggregate]) *CloudCostAggregateSet {
	if ccas == nil {
		return nil
	}

	result := ccas.Clone()
	result.filter(filters)

	return result
}

func (ccas *CloudCostAggregateSet) filter(filters filter.Filter[*CloudCostAggregate]) {
	if ccas == nil {
		return
	}

	if filters == nil {
		return
	}

	for name, cca := range ccas.CloudCostAggregates {
		if !filters.Matches(cca) {
			delete(ccas.CloudCostAggregates, name)
		}
	}
}

func (ccas *CloudCostAggregateSet) Insert(that *CloudCostAggregate) error {
	// Publicly, only allow Inserting as a basic operation (i.e. without causing
	// an aggregation on a property).
	return ccas.insertByProperty(that, nil)
}

func (ccas *CloudCostAggregateSet) insertByProperty(that *CloudCostAggregate, props []string) error {
	if ccas == nil {
		return fmt.Errorf("cannot insert into nil CloudCostAggregateSet")
	}

	if ccas.CloudCostAggregates == nil {
		ccas.CloudCostAggregates = map[string]*CloudCostAggregate{}
	}

	// Add the given CloudCostAggregate to the existing entry, if there is one;
	// otherwise just set directly into allocations
	if _, ok := ccas.CloudCostAggregates[that.Key(props)]; !ok {
		ccas.CloudCostAggregates[that.Key(props)] = that
	} else {
		ccas.CloudCostAggregates[that.Key(props)].add(that)
	}

	return nil
}

func (ccas *CloudCostAggregateSet) Clone() *CloudCostAggregateSet {
	aggs := make(map[string]*CloudCostAggregate, len(ccas.CloudCostAggregates))
	for k, v := range ccas.CloudCostAggregates {
		aggs[k] = v.Clone()
	}

	return &CloudCostAggregateSet{
		CloudCostAggregates: aggs,
		Integration:         ccas.Integration,
		LabelName:           ccas.LabelName,
		Window:              ccas.Window.Clone(),
	}
}

func (ccas *CloudCostAggregateSet) Equal(that *CloudCostAggregateSet) bool {
	if ccas.Integration != that.Integration {
		return false
	}

	if ccas.LabelName != that.LabelName {
		return false
	}

	if !ccas.Window.Equal(that.Window) {
		return false
	}

	if len(ccas.CloudCostAggregates) != len(that.CloudCostAggregates) {
		return false
	}

	for k, cca := range ccas.CloudCostAggregates {
		tcca, ok := that.CloudCostAggregates[k]
		if !ok {
			return false
		}
		if !cca.Equal(tcca) {
			return false
		}
	}

	return true
}

func (ccas *CloudCostAggregateSet) IsEmpty() bool {
	if ccas == nil {
		return true
	}

	if len(ccas.CloudCostAggregates) == 0 {
		return true
	}

	return false
}

func (ccas *CloudCostAggregateSet) Length() int {
	if ccas == nil {
		return 0
	}
	return len(ccas.CloudCostAggregates)
}

func (ccas *CloudCostAggregateSet) GetWindow() Window {
	return ccas.Window
}

func (ccas *CloudCostAggregateSet) Merge(that *CloudCostAggregateSet) (*CloudCostAggregateSet, error) {
	if ccas == nil || that == nil {
		return nil, fmt.Errorf("cannot merge nil CloudCostAggregateSets")
	}

	if that.IsEmpty() {
		return ccas.Clone(), nil
	}

	if !ccas.Window.Equal(that.Window) {
		return nil, fmt.Errorf("cannot merge CloudCostAggregateSets with different windows")
	}

	if ccas.LabelName != that.LabelName {
		return nil, fmt.Errorf("cannot merge CloudCostAggregateSets with different label names: '%s' != '%s'", ccas.LabelName, that.LabelName)
	}

	start, end := *ccas.Window.Start(), *ccas.Window.End()
	result := NewCloudCostAggregateSet(start, end)
	result.LabelName = ccas.LabelName

	for _, cca := range ccas.CloudCostAggregates {
		result.insertByProperty(cca, nil)
	}

	for _, cca := range that.CloudCostAggregates {
		result.insertByProperty(cca, nil)
	}

	return result, nil
}

func GetCloudCostAggregateSets(start, end time.Time, windowDuration time.Duration, integration string, labelName string) ([]*CloudCostAggregateSet, error) {
	windows, err := GetWindows(start, end, windowDuration)
	if err != nil {
		return nil, err
	}

	// Build slice of CloudCostAggregateSet to cover the range
	CloudCostAggregateSets := []*CloudCostAggregateSet{}
	for _, w := range windows {
		ccas := NewCloudCostAggregateSet(*w.Start(), *w.End())
		ccas.Integration = integration
		ccas.LabelName = labelName
		CloudCostAggregateSets = append(CloudCostAggregateSets, ccas)
	}
	return CloudCostAggregateSets, nil
}

// LoadCloudCostAggregateSets creates and loads CloudCostAggregates into provided CloudCostAggregateSets. This method makes it so
// that the input windows do not have to match the one day frame of the Athena queries. CloudCostAggregates being generated from a
// CUR which may be the identical except for the pricing model used (default, RI or savings plan)
// are accumulated here so that the resulting CloudCostAggregate with the 1d window has the correct price for the entire day.
func LoadCloudCostAggregateSets(itemStart time.Time, itemEnd time.Time, properties CloudCostAggregateProperties, K8sPercent, cost, credit float64, CloudCostAggregateSets []*CloudCostAggregateSet) {
	// Disperse cost of the current item across one or more CloudCostAggregates in
	// across each relevant CloudCostAggregateSet. Stop when the end of the current
	// block reaches the item's end time or the end of the range.
	for _, ccas := range CloudCostAggregateSets {
		pct := ccas.GetWindow().GetPercentInWindow(itemStart, itemEnd)

		// Insert an CloudCostAggregate with that cost into the CloudCostAggregateSet at the given index
		cca := &CloudCostAggregate{
			Properties:        properties,
			KubernetesPercent: K8sPercent * pct,
			Cost:              cost * pct,
			Credit:            credit * pct,
		}
		err := ccas.insertByProperty(cca, nil)
		if err != nil {
			log.Errorf("LoadCloudCostAggregateSets: failed to load CloudCostAggregate with key %s and window %s", cca.Key(nil), ccas.GetWindow().String())
		}
	}
}

type CloudCostAggregateSetRange struct {
	CloudCostAggregateSets []*CloudCostAggregateSet `json:"sets"`
	Window                 Window                   `json:"window"`
}

func (ccasr *CloudCostAggregateSetRange) Accumulate() (*CloudCostAggregateSet, error) {
	if ccasr == nil {
		return nil, errors.New("cannot accumulate a nil CloudCostAggregateSetRange")
	}

	if ccasr.Window.IsOpen() {
		return nil, fmt.Errorf("cannot accumulate a CloudCostAggregateSetRange with an open window: %s", ccasr.Window)
	}

	result := NewCloudCostAggregateSet(*ccasr.Window.Start(), *ccasr.Window.End())

	for _, ccas := range ccasr.CloudCostAggregateSets {
		for name, cca := range ccas.CloudCostAggregates {
			err := result.insertByProperty(cca.Clone(), ccas.AggregationProperties)
			if err != nil {
				return nil, fmt.Errorf("error accumulating CloudCostAggregateSetRange[%s][%s]: %s", ccas.Window.String(), name, err)
			}
		}
	}

	return result, nil
}
