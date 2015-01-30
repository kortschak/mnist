// Copyright ©2013 The bíogo.nn Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mnist

import (
	"testing"
)

func TestMnist(t *testing.T) {
	for _, test := range []struct {
		set  Set
		name string
		n    int
		rows int
		cols int
	}{
		{
			set:  Test,
			name: "Test",
			n:    10000,
			rows: 28,
			cols: 28,
		},
		{
			set:  Train,
			name: "Train",
			n:    60000,
			rows: 28,
			cols: 28,
		},
	} {
		if test.set.Len() != test.n {
			t.Errorf("Unexpected len for %q: got: %d want: %d", test.name, test.set.Len(), test.n)
		}
		if test.set.Rows() != test.rows {
			t.Errorf("Unexpected rows for %q: got: %d want: %d", test.name, test.set.Rows(), test.rows)
		}
		if test.set.Cols() != test.cols {
			t.Errorf("Unexpected cols for %q: got: %d want: %d", test.name, test.set.Cols(), test.cols)
		}
		if len(test.set.matrix) != test.n*test.rows*test.cols {
			t.Errorf("Unexpected matrix data length for %q: got: %d want: %d", test.name, len(test.set.matrix), test.n*test.rows*test.cols)
		}
		if len(test.set.labels) != test.n {
			t.Errorf("Unexpected number of labels for %q: got: %d want: %d", test.name, len(test.set.labels), test.n)
		}
	}
}
