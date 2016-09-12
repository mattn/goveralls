package main

import (
	"reflect"
	"testing"

	"golang.org/x/tools/cover"
)

func TestMergeProfs(t *testing.T) {
	tests := []struct {
		in   [][]*cover.Profile
		want []*cover.Profile
	}{
		// empty
		{in: nil, want: nil},
		// The number of profiles is 1
		{in: [][]*cover.Profile{[]*cover.Profile{{FileName: "name1"}}}, want: []*cover.Profile{{FileName: "name1"}}},
		// merge profile blocks
		{
			in: [][]*cover.Profile{
				[]*cover.Profile{}, // skip first empty profiles.
				[]*cover.Profile{
					{
						FileName: "name1",
						Blocks: []cover.ProfileBlock{
							cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 1},
						},
					},
					{
						FileName: "name2",
						Blocks: []cover.ProfileBlock{
							cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 0},
						},
					},
				},
				[]*cover.Profile{}, // skip first empty profiles.
				[]*cover.Profile{
					{
						FileName: "name1",
						Blocks: []cover.ProfileBlock{
							cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 1},
						},
					},
					{
						FileName: "name2",
						Blocks: []cover.ProfileBlock{
							cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 1},
						},
					},
				},
			},
			want: []*cover.Profile{
				{
					FileName: "name1",
					Blocks: []cover.ProfileBlock{
						cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 2},
					},
				},
				{
					FileName: "name2",
					Blocks: []cover.ProfileBlock{
						cover.ProfileBlock{StartLine: 1, StartCol: 1, Count: 1},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		if got := mergeProfs(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("mergeProfs(%#v) = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}
