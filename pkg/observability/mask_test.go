package observability

import (
	"strings"
	"testing"
)

func TestMaskMSISDN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "local format",
			in:   "call 09120001234 now",
			want: "call 0912***1234 now",
		},
		{
			name: "international plus format",
			in:   "recipient +989120001234",
			want: "recipient 0912***1234",
		},
		{
			name: "international 00 format",
			in:   "recipient 00989120001234",
			want: "recipient 0912***1234",
		},
		{
			name: "multiple numbers in one string",
			in:   "from 09120001111 to 09120002222",
			want: "from 0912***1111 to 0912***2222",
		},
		{
			name: "no phone number present",
			in:   "message_id=01JB0 status=DELIVERED",
			want: "message_id=01JB0 status=DELIVERED",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "leftmost valid-shaped match in a longer digit run is masked",
			in:   "ref 00912000123499120001234",
			want: "ref 00912***123499120001234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskMSISDN(tt.in)
			if got != tt.want {
				t.Errorf("MaskMSISDN(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskMSISDN_NeverPanics(_ *testing.T) {
	inputs := []string{"", "0", "09", "091200012", "09120001234890123", "+", "98", strings.Repeat("9", 50)}
	for _, in := range inputs {
		_ = MaskMSISDN(in)
	}
}
