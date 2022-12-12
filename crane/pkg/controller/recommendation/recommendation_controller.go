package recommendation

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
)

// RecommendationController is responsible for reconcile Recommendation
type RecommendationController struct {
	client.Client
	ConfigSet    *analysisv1alph1.ConfigSet
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	RestMapper   meta.RESTMapper
	ScaleClient  scale.ScalesGetter
	PredictorMgr predictormgr.Manager
	Provider     providers.History
}

func (c *RecommendationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got Recommendation %s", req.NamespacedName)

	recommendation := &analysisv1alph1.Recommendation{}
	err := c.Client.Get(ctx, req.NamespacedName, recommendation)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if recommendation.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// defaulting for TargetRef.Namespace
	if recommendation.Spec.TargetRef.Namespace == "" {
		recommendation.Spec.TargetRef.Namespace = recommendation.Namespace
	}

	newStatus := recommendation.Status.DeepCopy()

	updated, err := c.UpdateRecommendation(ctx, recommendation)
	if err != nil {
		c.Recorder.Event(recommendation, v1.EventTypeWarning, "FailedUpdateRecommendationValue", err.Error())
		msg := fmt.Sprintf("Failed to update recommendation value, Recommendation %s: %v", klog.KObj(recommendation), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedUpdateRecommendationValue", msg)
		return ctrl.Result{}, c.UpdateStatus(ctx, recommendation, newStatus)
	}

	if updated {
		c.Recorder.Event(recommendation, v1.EventTypeNormal, "UpdatedRecommendationValue", "")

		setReadyCondition(newStatus, metav1.ConditionTrue, "RecommendationReady", "Recommendation is ready")
		return ctrl.Result{}, c.UpdateStatus(ctx, recommendation, newStatus)
	}

	return ctrl.Result{}, nil
}

func (c *RecommendationController) UpdateStatus(ctx context.Context, recommendation *analysisv1alph1.Recommendation, newStatus *analysisv1alph1.RecommendationStatus) error {
	if !equality.Semantic.DeepEqual(&recommendation.Status, newStatus) {
		recommendation.Status = *newStatus
		timeNow := metav1.Now()
		recommendation.Status.LastUpdateTime = &timeNow

		err := c.Update(ctx, recommendation)
		if err != nil {
			c.Recorder.Event(recommendation, v1.EventTypeWarning, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, Recommendation %s error %v", klog.KObj(recommendation), err)
			return err
		}

		klog.Infof("Update Recommendation status successful, Recommendation %s", klog.KObj(recommendation))
	}

	return nil
}

func (c *RecommendationController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Recommendation{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}

func setReadyCondition(status *analysisv1alph1.RecommendationStatus, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for i := range status.Conditions {
		if status.Conditions[i].Type == "Ready" {
			status.Conditions[i].Status = conditionStatus
			status.Conditions[i].Reason = reason
			status.Conditions[i].Message = message
			status.Conditions[i].LastTransitionTime = metav1.Now()
			return
		}
	}
	status.Conditions = append(status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}
