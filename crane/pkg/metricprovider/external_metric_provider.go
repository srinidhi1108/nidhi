package metricprovider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
)

var _ provider.ExternalMetricsProvider = &ExternalMetricProvider{}

// ExternalMetricProvider implements ehpa external metric as external metric provider which now support cron metric
type ExternalMetricProvider struct {
	client        client.Client
	remoteAdapter *RemoteAdapter
	recorder      record.EventRecorder
	scaler        scale.ScalesGetter
	restMapper    meta.RESTMapper
}

// NewExternalMetricProvider returns an instance of ExternalMetricProvider
func NewExternalMetricProvider(client client.Client, remoteAdapter *RemoteAdapter, recorder record.EventRecorder, scaleClient scale.ScalesGetter, restMapper meta.RESTMapper) *ExternalMetricProvider {
	return &ExternalMetricProvider{
		client:        client,
		remoteAdapter: remoteAdapter,
		recorder:      recorder,
		scaler:        scaleClient,
		restMapper:    restMapper,
	}
}

const (
	// DefaultCronTargetMetricValue is used to construct a default external cron metric targetValue.
	// So the hpa may scale workload to DefaultCronTargetMetricValue. And finally scale replica depends on the HPA min max replica count the user set.
	DefaultCronTargetMetricValue int32 = 1
)

// GetExternalMetric get external metric according to metric type
func (p *ExternalMetricProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.Info(fmt.Sprintf("Get metric by selector for external metric, Info %v namespace %s metricSelector %s", info, namespace, metricSelector.String()))

	switch info.Metric {
	case known.MetricNameCron:
		return p.GetCronExternalMetrics(ctx, namespace, metricSelector, info)
	case known.MetricNamePrediction:
		predictions, err := GetPredictions(ctx, p.client, namespace, metricSelector)
		if err != nil {
			return nil, err
		}

		resourceIdentifier, found := metricSelector.RequiresExactMatch("resourceIdentifier")
		if !found {
			return nil, fmt.Errorf("failed get resourceIdentifier from metricSelector: [%v]", metricSelector)
		}

		for _, prediction := range predictions {
			timeSeries, err := utils.GetReadyPredictionMetric(info.Metric, resourceIdentifier, &prediction)
			if err != nil {
				return nil, err
			}

			// get the largest value from timeSeries
			// use the largest value will bring up the scaling up and defer the scaling down
			timestampStart := time.Now()
			timestampEnd := timestampStart.Add(time.Duration(prediction.Spec.PredictionWindowSeconds) * time.Second)
			largestMetricValue := &metricValue{}
			hasValidSample := false
			for _, v := range timeSeries.Samples {
				// exclude values that not in time range
				if v.Timestamp < timestampStart.Unix() || v.Timestamp > timestampEnd.Unix() {
					continue
				}

				valueFloat, err := strconv.ParseFloat(v.Value, 32)
				if err != nil {
					return nil, fmt.Errorf("failed to parse value to float: %v ", err)
				}
				if valueFloat > largestMetricValue.value {
					hasValidSample = true
					largestMetricValue.value = valueFloat
					largestMetricValue.timestamp = v.Timestamp
				}
			}

			if !hasValidSample {
				return nil, fmt.Errorf("TimeSeries is outdated, metric name %s", info.Metric)
			}

			klog.Infof("Provide external metric %s average value %f.", info.Metric, largestMetricValue.value)

			return &external_metrics.ExternalMetricValueList{Items: []external_metrics.ExternalMetricValue{
				{
					MetricName: info.Metric,
					Timestamp:  metav1.Now(),
					Value:      *resource.NewQuantity(int64(largestMetricValue.value), resource.DecimalSI),
				},
			}}, nil
		}
	default:
		if p.remoteAdapter != nil {
			return p.remoteAdapter.GetExternalMetric(ctx, namespace, metricSelector, info)
		} else {
			return nil, apiErrors.NewServiceUnavailable("not supported")
		}
	}

	return nil, apiErrors.NewServiceUnavailable("metric not found")
}

// GetCronExternalMetrics get desired metric value from cron spec
func (p *ExternalMetricProvider) GetCronExternalMetrics(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.Infof("Get cron metric %s by selector", info.Metric)

	var ehpaList autoscalingapi.EffectiveHorizontalPodAutoscalerList
	err := p.client.List(context.TODO(), &ehpaList)
	if err != nil {
		klog.Errorf("Failed to list ehpa: %v", err)
		return &external_metrics.ExternalMetricValueList{}, err
	}

	targetKind, bl := metricSelector.RequiresExactMatch("targetKind")
	if !bl {
		return &external_metrics.ExternalMetricValueList{}, fmt.Errorf("get cron external metrics, metricSelector: [%v] target [%s] not matched", metricSelector, "targetKind")
	}
	targetName, bl := metricSelector.RequiresExactMatch("targetName")
	if !bl {
		return &external_metrics.ExternalMetricValueList{}, fmt.Errorf("get cron external metrics, metricSelector: [%v] target [%s] not matched", metricSelector, "targetName")
	}
	targetNamespace, bl := metricSelector.RequiresExactMatch("targetNamespace")
	if !bl {
		return &external_metrics.ExternalMetricValueList{}, fmt.Errorf("get cron external metrics, metricSelector: [%v] target [%s] not matched", metricSelector, "targetNamespace")
	}

	var ehpa autoscalingapi.EffectiveHorizontalPodAutoscaler
	for _, item := range ehpaList.Items {
		if utils.IsEHPACronEnabled(&item) && item.Spec.ScaleTargetRef.Kind == targetKind && item.Spec.ScaleTargetRef.Name == targetName && item.Namespace == targetNamespace {
			ehpa = item
		}
	}

	// Find the cron metric scaler
	cronScalers := GetCronScalersForEHPA(&ehpa)
	var activeScalers []*CronScaler
	var errs []error
	for _, cronScaler := range cronScalers {
		isActive, err := cronScaler.IsActive(ctx, time.Now())
		if err != nil {
			errs = append(errs, err)
		}
		if isActive {
			activeScalers = append(activeScalers, cronScaler)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%v", errs)
	}

	// Set default replicas same with minReplicas of ehpa
	replicas := *ehpa.Spec.MinReplicas
	// we use the largest targetReplicas specified in cron spec.
	for _, activeScaler := range activeScalers {
		if activeScaler.TargetSize() >= replicas {
			replicas = activeScaler.TargetSize()
		}
	}

	return &external_metrics.ExternalMetricValueList{Items: []external_metrics.ExternalMetricValue{
		{
			MetricName: info.Metric,
			Timestamp:  metav1.Now(),
			Value:      *resource.NewQuantity(int64(replicas), resource.DecimalSI),
		},
	}}, nil
}

// ListAllExternalMetrics return external cron metrics
// Fetch metrics from cache directly to avoid the performance issue for apiserver when the metrics is large, because this api is called frequently.
func (p *ExternalMetricProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	klog.Info("List all external metrics")

	var metricInfos []provider.ExternalMetricInfo
	//add cron metric
	metricInfos = append(metricInfos, provider.ExternalMetricInfo{Metric: known.MetricNameCron})
	//add prediction metric
	metricInfos = append(metricInfos, provider.ExternalMetricInfo{Metric: known.MetricNamePrediction})

	if p.remoteAdapter != nil {
		metricInfos = append(metricInfos, p.remoteAdapter.ListAllExternalMetrics()...)
	}
	return metricInfos
}

func IsLocalExternalMetric(metricInfo provider.ExternalMetricInfo, client client.Client) bool {
	switch metricInfo.Metric {
	case known.MetricNameCron, known.MetricNamePrediction:
		return true
	}
	return false
}

func CronEnabled(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	return len(ehpa.Spec.Crons) > 0
}

// EHPACronMetricName return the hpa cron external metric name from ehpa cron scale spec
// construct the cron metric name by ehpa namespace, name, cron name, cron timezone, cron start, cron end
// make sure each ehpa cron scale metric name is unique.
func EHPACronMetricName(namespace, name string, cronScale autoscalingapi.CronSpec) string {
	// same timezone return different cases when in different machine. transfer to lower case
	timezone := GetCronScaleLocation(cronScale)
	// metric name must be lower case, can not container upper case: https://github.com/kubernetes/kubernetes/issues/72996
	return NormalizeString(strings.ToLower(fmt.Sprintf("cron-%v-%v-%v-%v-%v-%v", namespace, name, cronScale.Name, strings.ToLower(timezone.String()), shapeCronTimeFormat(cronScale.Start), shapeCronTimeFormat(cronScale.End))))
}

// GetCronScaleLocation return the cronScale location, default is UTC when it is not specified in spec
func GetCronScaleLocation(cronScale autoscalingapi.CronSpec) *time.Location {
	timezone := time.Local
	var err error
	if cronScale.TimeZone != nil {
		timezone, err = time.LoadLocation(*cronScale.TimeZone)
		if err != nil {
			klog.Errorf("Failed to parse timezone %v, use default %+v, err: %v", *cronScale.TimeZone, timezone, err)
		}
	}
	return timezone
}

func GetCronScalersForEHPA(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) []*CronScaler {
	var scalers []*CronScaler
	for _, cronScale := range ehpa.Spec.Crons {
		cronMetricName := EHPACronMetricName(ehpa.Namespace, ehpa.Name, cronScale)
		scalers = append(scalers, NewCronScaler(&CronTrigger{
			Name:     cronMetricName,
			Location: GetCronScaleLocation(cronScale),
			Start:    cronScale.Start,
			End:      cronScale.End,
		}, ehpa, cronScale.TargetReplicas))
	}
	return scalers
}

func shapeCronTimeFormat(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "*", "x")
	s = strings.ReplaceAll(s, "/", "sl")
	s = strings.ReplaceAll(s, "?", "qm")
	return s
}

func NormalizeString(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "%", "-")
	return s
}

type CronScaler struct {
	trigger        *CronTrigger
	ref            *autoscalingapi.EffectiveHorizontalPodAutoscaler
	targetReplicas int32
}

func NewCronScaler(trigger *CronTrigger, ref *autoscalingapi.EffectiveHorizontalPodAutoscaler, targetReplicas int32) *CronScaler {
	return &CronScaler{
		trigger:        trigger,
		ref:            ref,
		targetReplicas: targetReplicas,
	}
}

func (cs *CronScaler) IsActive(ctx context.Context, now time.Time) (bool, error) {
	return cs.trigger.IsActive(ctx, now)
}

func (cs *CronScaler) Name() string {
	return cs.trigger.Name
}

func (cs *CronScaler) TargetSize() int32 {
	return cs.targetReplicas
}

func GetPredictions(ctx context.Context, kubeclient client.Client, namespace string, metricSelector labels.Selector) ([]predictionapi.TimeSeriesPrediction, error) {
	labelSelector, err := labels.ConvertSelectorToLabelsMap(metricSelector.String())
	if err != nil {
		klog.Error(err, "Failed to convert metric selectors to labels")
		return nil, err
	}

	matchingLabels := client.MatchingLabels(map[string]string{"app.kubernetes.io/managed-by": known.EffectiveHorizontalPodAutoscalerManagedBy})
	// merge metric selectors
	for key, value := range labelSelector {
		switch key {
		case "targetKind":
			matchingLabels["app.kubernetes.io/target-kind"] = value
		case "targetNamespace":
			matchingLabels["app.kubernetes.io/target-namespace"] = value
		case "targetName":
			matchingLabels["app.kubernetes.io/target-name"] = value
		}
	}

	predictionList := &predictionapi.TimeSeriesPredictionList{}
	opts := []client.ListOption{
		matchingLabels,
		client.InNamespace(namespace),
	}
	err = kubeclient.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get TimeSeriesPrediction when get custom metric ")
	} else if len(predictionList.Items) == 0 {
		return nil, fmt.Errorf("there is no TimeSeriesPrediction match the selector %s ", metricSelector.String())
	}

	return predictionList.Items, nil
}
