package supportticket

import "testing"

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name             string
		page, pageSize   int
		wantPage, wantPS int
	}{
		{"zero values default", 0, 0, 1, 20},
		{"negative values default", -5, -1, 1, 20},
		{"valid values pass through", 3, 50, 3, 50},
		{"pageSize over 100 defaults", 1, 200, 1, 20},
		{"page 1 pageSize 1", 1, 1, 1, 1},
		{"boundary pageSize 100", 1, 100, 1, 100},
		{"boundary pageSize 101 defaults", 1, 101, 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPS := normalizePagination(tt.page, tt.pageSize)
			if gotPage != tt.wantPage {
				t.Errorf("normalizePagination(%d, %d) page = %d, want %d", tt.page, tt.pageSize, gotPage, tt.wantPage)
			}
			if gotPS != tt.wantPS {
				t.Errorf("normalizePagination(%d, %d) pageSize = %d, want %d", tt.page, tt.pageSize, gotPS, tt.wantPS)
			}
		})
	}
}
