package stk

import (
	"slices"
	"testing"
	"time"
)

func TestDateTimeZone(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want []byte
	}{
		{
			name: "positive offset",
			t:    time.Date(2026, 7, 4, 20, 15, 9, 0, time.FixedZone("test", 8*60*60)),
			want: []byte{0x62, 0x70, 0x40, 0x02, 0x51, 0x90, 0x23},
		},
		{
			name: "negative offset",
			t:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("test", -5*60*60)),
			want: []byte{0x62, 0x10, 0x20, 0x30, 0x40, 0x50, 0x02 | 0x08},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DateTimeZone(tt.t)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("DateTimeZone() = % X, want % X", got, tt.want)
			}
		})
	}
}
