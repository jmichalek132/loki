package iter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/grafana/loki/v3/pkg/util/constants"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		iterators []Iterator
		expected  []patternSample
	}{
		{
			name:      "Empty iterators",
			iterators: []Iterator{},
			expected:  nil,
		},
		{
			name: "Merge single iterator",
			iterators: []Iterator{
				NewSlice("a", constants.LogLevelInfo, []logproto.PatternSample{
					{Timestamp: 10, Value: 2}, {Timestamp: 20, Value: 4}, {Timestamp: 30, Value: 6},
				}),
			},
			expected: []patternSample{
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 10, Value: 2}},
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 20, Value: 4}},
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 30, Value: 6}},
			},
		},
		{
			name: "Merge multiple iterators",
			iterators: []Iterator{
				NewSlice("a", constants.LogLevelInfo, []logproto.PatternSample{{Timestamp: 10, Value: 2}, {Timestamp: 30, Value: 6}}),
				NewSlice("b", constants.LogLevelInfo, []logproto.PatternSample{{Timestamp: 20, Value: 4}, {Timestamp: 40, Value: 8}}),
			},
			expected: []patternSample{
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 10, Value: 2}},
				{"b", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 20, Value: 4}},
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 30, Value: 6}},
				{"b", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 40, Value: 8}},
			},
		},
		{
			name: "Merge multiple iterators with similar samples",
			iterators: []Iterator{
				NewSlice("a", constants.LogLevelInfo, []logproto.PatternSample{{Timestamp: 10, Value: 2}, {Timestamp: 30, Value: 6}}),
				NewSlice("a", constants.LogLevelInfo, []logproto.PatternSample{{Timestamp: 10, Value: 2}, {Timestamp: 30, Value: 6}}),
				NewSlice("b", constants.LogLevelInfo, []logproto.PatternSample{{Timestamp: 20, Value: 4}, {Timestamp: 40, Value: 8}}),
			},
			expected: []patternSample{
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 10, Value: 4}},
				{"b", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 20, Value: 4}},
				{"a", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 30, Value: 12}},
				{"b", constants.LogLevelInfo, logproto.PatternSample{Timestamp: 40, Value: 8}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := NewMerge(tt.iterators...)
			defer it.Close()

			var result []patternSample
			for it.Next() {
				result = append(result, patternSample{it.Pattern(), constants.LogLevelInfo, it.At()})
			}

			require.Equal(t, tt.expected, result)
		})
	}
}
