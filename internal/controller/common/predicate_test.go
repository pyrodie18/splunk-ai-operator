package common

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestLabelChangedPredicate(t *testing.T) {
	g := NewWithT(t)
	pred := LabelChangedPredicate()

	t.Run("Create event returns true", func(t *testing.T) {
		e := event.CreateEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Create(e)).To(BeTrue())
	})

	t.Run("Update with label change returns true", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key": "old"},
			},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key": "new"},
			},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeTrue())
	})

	t.Run("Update with no label change returns false", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key": "value"},
			},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"key": "value"},
			},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeFalse())
	})

	t.Run("Delete event returns true", func(t *testing.T) {
		e := event.DeleteEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Delete(e)).To(BeTrue())
	})

	t.Run("Generic event returns true", func(t *testing.T) {
		e := event.GenericEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Generic(e)).To(BeTrue())
	})
}

func TestGenerationChangedPredicate(t *testing.T) {
	g := NewWithT(t)
	pred := GenerationChangedPredicate()

	t.Run("Create event returns true", func(t *testing.T) {
		e := event.CreateEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Create(e)).To(BeTrue())
	})

	t.Run("Update with generation change returns true", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Generation: 1},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Generation: 2},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeTrue())
	})

	t.Run("Update with no generation change returns false", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Generation: 1},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Generation: 1},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeFalse())
	})

	t.Run("Delete event returns true", func(t *testing.T) {
		e := event.DeleteEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Delete(e)).To(BeTrue())
	})

	t.Run("Generic event returns true", func(t *testing.T) {
		e := event.GenericEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Generic(e)).To(BeTrue())
	})
}

func TestAnnotationChangedPredicate(t *testing.T) {
	g := NewWithT(t)
	pred := AnnotationChangedPredicate()

	t.Run("Create event returns true", func(t *testing.T) {
		e := event.CreateEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Create(e)).To(BeTrue())
	})

	t.Run("Update with annotation change returns true", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"key": "old"},
			},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"key": "new"},
			},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeTrue())
	})

	t.Run("Update with no annotation change returns false", func(t *testing.T) {
		oldObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"key": "value"},
			},
		}
		newObj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"key": "value"},
			},
		}
		e := event.UpdateEvent{
			ObjectOld: oldObj,
			ObjectNew: newObj,
		}
		g.Expect(pred.Update(e)).To(BeFalse())
	})

	t.Run("Delete event returns true", func(t *testing.T) {
		e := event.DeleteEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Delete(e)).To(BeTrue())
	})

	t.Run("Generic event returns true", func(t *testing.T) {
		e := event.GenericEvent{
			Object: &corev1.Pod{},
		}
		g.Expect(pred.Generic(e)).To(BeTrue())
	})
}

func TestStringInSlice(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns true when string is in slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		g.Expect(stringInSlice("two", slice)).To(BeTrue())
	})

	t.Run("returns false when string is not in slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		g.Expect(stringInSlice("four", slice)).To(BeFalse())
	})

	t.Run("returns false for empty slice", func(t *testing.T) {
		slice := []string{}
		g.Expect(stringInSlice("test", slice)).To(BeFalse())
	})
}
