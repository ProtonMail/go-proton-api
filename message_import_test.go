package proton

import (
	"reflect"
	"testing"
)

func Test_chunkSized(t *testing.T) {
	type args struct {
		vals    []int
		maxLen  int
		maxSize int
		getSize func(int) int
	}

	tests := []struct {
		name string
		args args
		want [][]int
	}{
		{
			name: "limit by length",
			args: args{
				vals:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				maxLen:  3, // Split into chunks of at most 3
				maxSize: 100,
				getSize: func(i int) int { return i },
			},
			want: [][]int{
				{1, 2, 3},
				{4, 5, 6},
				{7, 8, 9},
				{10},
			},
		},
		{
			name: "limit by size",
			args: args{
				vals:    []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2},
				maxLen:  100,
				maxSize: 5, // Split into chunks of at most 5
				getSize: func(i int) int { return i },
			},
			want: [][]int{
				{1, 1, 1, 1, 1},
				{2, 2},
				{2, 2},
				{2},
			},
		},
		{
			name: "single values larger than max",
			args: args{
				vals:    []int{1, 2, 3, 100, 200, 1, 2, 3, 4},
				maxLen:  100,
				maxSize: 10, // Split into chunks of at most 10, but let single values larger than max through
				getSize: func(i int) int { return i },
			},
			want: [][]int{
				{1, 2, 3}, // Attempting to add 100 to this chunk would exceed the max size
				{100},     // Single value larger than max
				{200},     // Single value larger than max
				{1, 2, 3, 4},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chunkSized(tt.args.vals, tt.args.maxLen, tt.args.maxSize, tt.args.getSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chunkSized() = %v, want %v", got, tt.want)
			}
		})
	}
}
