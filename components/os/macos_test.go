package os

import (
	"testing"

	"github.com/DataDog/test-infra-definitions/resources/aws"
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

		env := &aws.Environment{}
		testos := NewUnix(env)

		var res bool
		for _, test := range tests {

			res = testos.CheckIsAbsPath(test.path)
			if res != test.want {
				t.Errorf("CheckIsAbsPath(\"%s\") evaluated wrong - want: %t, got: %t", test.path, res, test.want)
			}
		}
	})
}
