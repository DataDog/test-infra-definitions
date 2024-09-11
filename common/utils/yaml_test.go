package utils

import (
	"testing"
)

func TestMergeYAML(t *testing.T) {
	tests := map[string]struct {
		oldValues      string
		newValues      string
		expectedResult string
		expectError    bool
	}{
		"no new values":            {oldValues: "a: 1\nb: 2", newValues: "", expectedResult: "a: 1\nb: 2", expectError: false},
		"no old values":            {oldValues: "", newValues: "a: 1\nb: 2", expectedResult: "a: 1\nb: 2", expectError: false},
		"old value not valid yaml": {oldValues: "- a:b:", newValues: "a: 1\nb: 2", expectedResult: "", expectError: true},
		"new value not valid yaml": {oldValues: "a: 1\nb: 2", newValues: "- a:b:", expectedResult: "", expectError: true},
		"golden path":              {oldValues: "a: 1", newValues: "b: 2", expectedResult: "a: 1\nb: 2\n", expectError: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := MergeYAML(tc.oldValues, tc.newValues)
			if tc.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if got != tc.expectedResult {
				t.Fatalf("expected result %v, got %v", tc.expectedResult, got)
			}
		})
	}
}
