package pdf417

import "testing"

// TestRowIndicatorCodewords pins getLeftCodeWord and getRightCodeWord
// against hand-computed values for every table as a regression test.
func TestRowIndicatorCodewords(t *testing.T) {
	// USS PDF 417: Sec. 2.2.3: a PDF417 symbol may have as few as
	// 3 or as many as 90 rows and one to thirty columns of data
	// symbol characters.
	const (
		testMinRows = 3
		testMaxRows = 90
		testMinCols = 1
		testMaxCols = 30
	)
	tests := []struct {
		name          string
		rowNum        int
		rows          int
		columns       int
		securityLevel byte
		wantLeft      int
		wantRight     int
	}{
		{"table0/group0", 0, 17, 6, 3, 5, 5},
		{"table1/group0", 1, 17, 6, 3, 10, 5},
		{"table2/group0", 2, 17, 6, 3, 5, 10},
		{"table0/group1", 3, 17, 6, 3, 35, 35},
		{"table1/group1", 4, 17, 6, 3, 40, 35},
		{"table2/group1", 5, 17, 6, 3, 35, 40},
		{"table1/group5", 16, 17, 6, 3, 160, 155},
		{"table0/maxParams", 0, testMaxRows, testMaxCols, 8, 29, 29},
		{"table1/maxParams", 1, testMaxRows, testMaxCols, 8, 26, 29},
		{"table2/maxParams", 2, testMaxRows, testMaxCols, 8, 29, 26},
		{"table0/lastRow", testMaxRows - 3, testMaxRows, testMaxCols, 8, 899, 899},
		{"table1/lastRow", testMaxRows - 2, testMaxRows, testMaxCols, 8, 896, 899},
		{"table2/lastRow", testMaxRows - 1, testMaxRows, testMaxCols, 8, 899, 896},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLeftCodeWord(tt.rowNum, tt.rows, tt.columns, tt.securityLevel); got != tt.wantLeft {
				t.Errorf("getLeftCodeWord(rowNum=%d, rows=%d, columns=%d, securityLevel=%d) = %d, want %d",
					tt.rowNum, tt.rows, tt.columns, tt.securityLevel, got, tt.wantLeft)
			}
			if got := getRightCodeWord(tt.rowNum, tt.rows, tt.columns, tt.securityLevel); got != tt.wantRight {
				t.Errorf("getRightCodeWord(rowNum=%d, rows=%d, columns=%d, securityLevel=%d) = %d, want %d",
					tt.rowNum, tt.rows, tt.columns, tt.securityLevel, got, tt.wantRight)
			}
		})
	}
}
