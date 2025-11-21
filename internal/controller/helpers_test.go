package controller

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestContainsString(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns true when string is in slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		g.Expect(containsString(slice, "two")).To(BeTrue())
	})

	t.Run("returns false when string is not in slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		g.Expect(containsString(slice, "four")).To(BeFalse())
	})

	t.Run("returns false for empty slice", func(t *testing.T) {
		slice := []string{}
		g.Expect(containsString(slice, "test")).To(BeFalse())
	})

	t.Run("returns true for first element", func(t *testing.T) {
		slice := []string{"first", "second", "third"}
		g.Expect(containsString(slice, "first")).To(BeTrue())
	})

	t.Run("returns true for last element", func(t *testing.T) {
		slice := []string{"first", "second", "third"}
		g.Expect(containsString(slice, "third")).To(BeTrue())
	})
}

func TestRemoveString(t *testing.T) {
	g := NewWithT(t)

	t.Run("removes string from middle of slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		result := removeString(slice, "two")
		g.Expect(result).To(Equal([]string{"one", "three"}))
	})

	t.Run("removes string from beginning of slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		result := removeString(slice, "one")
		g.Expect(result).To(Equal([]string{"two", "three"}))
	})

	t.Run("removes string from end of slice", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		result := removeString(slice, "three")
		g.Expect(result).To(Equal([]string{"one", "two"}))
	})

	t.Run("returns unchanged slice when string not found", func(t *testing.T) {
		slice := []string{"one", "two", "three"}
		result := removeString(slice, "four")
		g.Expect(result).To(Equal([]string{"one", "two", "three"}))
	})

	t.Run("returns empty slice when removing from single element", func(t *testing.T) {
		slice := []string{"only"}
		result := removeString(slice, "only")
		g.Expect(result).To(BeEmpty())
	})

	t.Run("handles empty slice", func(t *testing.T) {
		slice := []string{}
		result := removeString(slice, "test")
		g.Expect(result).To(BeEmpty())
	})
}
