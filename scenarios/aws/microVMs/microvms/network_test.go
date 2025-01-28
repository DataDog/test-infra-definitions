package microvms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArpLine(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		want      dhcpLease
		wantError bool
	}{
		{
			name: "valid line",
			line: "? (10.211.55.4) at 0:1c:42:a:70:b on bridge100 ifscope [bridge]",
			want: dhcpLease{
				ip:  "10.211.55.4",
				mac: "0:1c:42:a:70:b",
			},
		},
		{
			name: "line with hostname",
			line: "agent-dev-ubuntu-22.shared (10.211.55.4) at 0:1c:42:a:70:b on bridge100 ifscope [bridge]",
			want: dhcpLease{
				ip:  "10.211.55.4",
				mac: "0:1c:42:a:70:b",
			},
		},
		{
			name:      "invalid mac address",
			line:      "? (1.2.3.4) at 0:1c:42:a:70 on bridge100 ifscope [bridge]",
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lease, err := parseArpLine(tc.line)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, lease)
			}
		})
	}
}

func TestNormalizeMAC(t *testing.T) {
	cases := []struct {
		name string
		mac  string
		want string
	}{
		{
			name: "lowercase",
			mac:  "0:1c:42:a:70:b",
			want: "00:1c:42:0a:70:0b",
		},
		{
			name: "uppercase",
			mac:  "0:1C:42:A:70:B",
			want: "00:1c:42:0a:70:0b",
		},
		{
			name: "invalid",
			mac:  "0:1c:42:a:70",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := normalizeMAC(tc.mac)
			if tc.want == "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, actual)
			}
		})
	}

}
