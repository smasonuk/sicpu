package grid

import "testing"

func TestGetGridCoords(t *testing.T) {
	tests := []struct {
		index int
		cols  int
		wantX int
		wantY int
	}{
		// 64 cols (Standard)
		{0, 64, 0, 0},
		{1, 64, 1, 0},
		{63, 64, 63, 0},
		{64, 64, 0, 1},
		{65, 64, 1, 1},
		{127, 64, 63, 1},
		{128, 64, 0, 2},
		{1023, 64, 63, 15},

		// 32 cols (Low Res)
		{0, 32, 0, 0},
		{31, 32, 31, 0},
		{32, 32, 0, 1},
		{63, 32, 31, 1},
		{1023, 32, 31, 31},
	}

	for _, tc := range tests {
		gotX, gotY := GetGridCoords(tc.index, tc.cols)
		if gotX != tc.wantX || gotY != tc.wantY {
			t.Errorf("GetGridCoords(%d, %d) = (%d, %d); want (%d, %d)", tc.index, tc.cols, gotX, gotY, tc.wantX, tc.wantY)
		}
	}
}
