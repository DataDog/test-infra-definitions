package os

import (
	"testing"

	"github.com/DataDog/test-infra-definitions/resources/azure"
)

func TestWindowsOS(t *testing.T) {
	t.Run("CheckIsAbsPath evaluates windows paths", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			// test cases match stdlib filepath.IsAbs test cases
			// https://github.com/golang/go/blob/master/src/path/filepath/path_test.go#L1048-L1063
			{`C:\`, true},
			{`c\`, false},
			{`c::`, false},
			{`c:`, false},
			{`/`, false},
			{`\`, false},
			{`\Windows`, false},
			{`c:a\b`, false},
			{`c:\a\b`, true},
			{`c:/a/b`, true},
			{`\\host\share`, true},
			{`\\host\share\`, true},
			{`\\host\share\foo`, true},
			{`//host/share/foo/bar`, true},
		}

		env := &azure.Environment{}
		testos := NewWindows(env)

		var res bool
		for _, test := range tests {

			res = testos.CheckIsAbsPath(test.path)
			if res != test.want {
				t.Errorf("CheckIsAbsPath(\"%s\") evaluated wrong - want: %t, got: %t", test.path, res, test.want)
			}
		}
	})
}
