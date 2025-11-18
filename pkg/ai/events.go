package ai_platform

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// EventHelper provides helper methods for emitting events with state transition detection
type EventHelper struct {
	recorder record.EventRecorder
}

// NewEventHelper creates a new EventHelper
func NewEventHelper(recorder record.EventRecorder) *EventHelper {
	return &EventHelper{recorder: recorder}
}

// EmitStageEvent emits an event for a reconciliation stage
// Only emits events on transitions (success after failure or failure after success)
func (h *EventHelper) EmitStageEvent(object runtime.Object, stageName string, err error, prevConditions []metav1.Condition) {
	if err != nil {
		// Check if previous state was success
		prevSuccess := false
		for _, cond := range prevConditions {
			if cond.Type == stageName+"Ready" && cond.Status == metav1.ConditionTrue {
				prevSuccess = true
				break
			}
		}
		// Only emit if transitioning from success to failure or first time
		if prevSuccess || len(prevConditions) == 0 {
			h.recorder.Eventf(object, v1.EventTypeWarning, stageName+"Failed",
				"Stage %s failed: %v", stageName, err)
		}
	} else {
		// Check if previous state was failure
		prevFailed := false
		for _, cond := range prevConditions {
			if cond.Type == stageName+"Ready" && cond.Status == metav1.ConditionFalse {
				prevFailed = true
				break
			}
		}
		// Only emit if transitioning from failure to success
		if prevFailed {
			h.recorder.Eventf(object, v1.EventTypeNormal, stageName+"Succeeded",
				"Stage %s completed successfully", stageName)
		}
	}
}

// EmitLifecycleEvent emits a lifecycle event (always emitted)
func (h *EventHelper) EmitLifecycleEvent(object runtime.Object, reason, message string) {
	h.recorder.Event(object, v1.EventTypeNormal, reason, message)
}

// EmitErrorEvent emits an error event (always emitted)
func (h *EventHelper) EmitErrorEvent(object runtime.Object, reason, message string) {
	h.recorder.Event(object, v1.EventTypeWarning, reason, message)
}

// EmitTransitionEvent emits an event only if the condition status changed
func (h *EventHelper) EmitTransitionEvent(object runtime.Object, conditionType string, newStatus metav1.ConditionStatus, prevConditions []metav1.Condition, message string) {
	prevStatus := metav1.ConditionUnknown
	for _, cond := range prevConditions {
		if cond.Type == conditionType {
			prevStatus = cond.Status
			break
		}
	}

	// Emit event only if status changed
	if newStatus != prevStatus {
		eventType := v1.EventTypeNormal
		if newStatus == metav1.ConditionFalse {
			eventType = v1.EventTypeWarning
		}
		h.recorder.Event(object, eventType, conditionType, message)
	}
}
