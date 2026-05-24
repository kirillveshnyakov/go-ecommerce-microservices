package outbox

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{
			name: "undefined",
			kind: KindUndefined,
			want: "undefined",
		},
		{
			name: "notification",
			kind: KindNotification,
			want: "notification",
		},
		{
			name: "unknown",
			kind: Kind(100),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.kind.String()
			require.Equal(t, tt.want, got)
		})
	}
}
