package bot

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "н/д"},
		{30 * time.Second, "1м"},
		{50 * time.Minute, "50м"},
		{2*time.Hour + 27*time.Minute, "2ч 27м"},
		{-(2*time.Hour + 27*time.Minute), "2ч 27м"},
		{3*24*time.Hour + 52*time.Minute, "3д 0ч 52м"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v)=%q want %q", c.d, got, c.want)
		}
	}
}
