package os

import (
	"testing"
)

func TestMacOS(t *testing.T) {
	t.Run("CheckIsAbsPath evaluates unix paths", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"", false},
			{"/", true},
			{"/usr/bin/gcc", true},
			{"..", false},
			{"/a/../bb", true},
			{".", false},
			{"./", false},
			{"lala", false},
		}

		testos := NewUnix()

		var res bool
		for _, test := range tests {

			res = testos.CheckIsAbsPath(test.path)
			if res != test.want {
				t.Errorf("CheckIsAbsPath(\"%s\") evaluated wrong - want: %t, got: %t", test.path, res, test.want)
			}
		}
	})
}
