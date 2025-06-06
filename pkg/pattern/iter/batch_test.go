package iter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/grafana/loki/v3/pkg/util/constants"
)

func TestReadBatch(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		level     string
		samples   []logproto.PatternSample
		batchSize int
		expected  *logproto.QueryPatternsResponse
	}{
		{
			name:      "ReadBatch empty iterator",
			pattern:   "foo",
			level:     constants.LogLevelInfo,
			samples:   []logproto.PatternSample{},
			batchSize: 2,
			expected: &logproto.QueryPatternsResponse{
				Series: []*logproto.PatternSeries{},
			},
		},
		{
			name:      "ReadBatch less than batchSize",
			pattern:   "foo",
			level:     constants.LogLevelInfo,
			samples:   []logproto.PatternSample{{Timestamp: 10, Value: 2}, {Timestamp: 20, Value: 4}, {Timestamp: 30, Value: 6}},
			batchSize: 2,
			expected: &logproto.QueryPatternsResponse{
				Series: []*logproto.PatternSeries{
					{
						Pattern: "foo",
						Level:   constants.LogLevelInfo,
						Samples: []*logproto.PatternSample{
							{Timestamp: 10, Value: 2},
							{Timestamp: 20, Value: 4},
						},
					},
				},
			},
		},
		{
			name:      "ReadBatch more than batchSize",
			pattern:   "foo",
			level:     constants.LogLevelInfo,
			samples:   []logproto.PatternSample{{Timestamp: 10, Value: 2}, {Timestamp: 20, Value: 4}, {Timestamp: 30, Value: 6}},
			batchSize: 4,
			expected: &logproto.QueryPatternsResponse{
				Series: []*logproto.PatternSeries{
					{
						Pattern: "foo",
						Level:   constants.LogLevelInfo,
						Samples: []*logproto.PatternSample{
							{Timestamp: 10, Value: 2},
							{Timestamp: 20, Value: 4},
							{Timestamp: 30, Value: 6},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := NewSlice(tt.pattern, tt.level, tt.samples)
			got, err := ReadBatch(it, tt.batchSize)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}
