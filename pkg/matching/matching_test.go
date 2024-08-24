package matching

import (
	"testing"

	"github.com/matryer/is"
)

func compareSlices(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, j := range a {
		if j != b[i] {
			return false
		}
	}
	return true
}

// The test cases numbers correspond to the python implementation

func TestMaxWeightMatching10(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{}))
	is.True(weight == 0)
}

func TestMaxWeightMatching11(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(0, 1, 1)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{1, 0}))
	is.True(weight == 1)
}

func TestMaxWeightMatching12(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 10), NewEdge(2, 3, 11)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, -1, 3, 2}))
	is.True(weight == 11)
}

func TestMaxWeightMatching13(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 5), NewEdge(2, 3, 11), NewEdge(3, 4, 5)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, -1, 3, 2, -1}))
	is.True(weight == 11)
}

func TestMaxWeightMatching14(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 5), NewEdge(2, 3, 11), NewEdge(3, 4, 5)}, true)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3}))
	is.True(weight == 10)
}

func TestMaxWeightMatching16(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 2), NewEdge(1, 3, -2), NewEdge(2, 3, 1), NewEdge(2, 4, -1), NewEdge(3, 4, -6)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, -1, -1}))
	is.True(weight == 2)
	mates, weight, err = maxWeightMatching([]*Edge{NewEdge(1, 2, 2), NewEdge(1, 3, -2), NewEdge(2, 3, 1), NewEdge(2, 4, -1), NewEdge(3, 4, -6)}, true)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 3, 4, 1, 2}))
	is.True(weight == -3)
}

func TestMaxWeightMatching20(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 8), NewEdge(1, 3, 9), NewEdge(2, 3, 10), NewEdge(3, 4, 7)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3}))
	is.True(weight == 15)
	mates, weight, err = maxWeightMatching([]*Edge{NewEdge(1, 2, 8), NewEdge(1, 3, 9), NewEdge(2, 3, 10), NewEdge(3, 4, 7), NewEdge(1, 6, 5), NewEdge(4, 5, 6)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 21)
}

func TestMaxWeightMatching21(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 9), NewEdge(1, 3, 8), NewEdge(2, 3, 10), NewEdge(1, 4, 5), NewEdge(4, 5, 4), NewEdge(1, 6, 3)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 17)
	mates, weight, err = maxWeightMatching([]*Edge{NewEdge(1, 2, 9), NewEdge(1, 3, 8), NewEdge(2, 3, 10), NewEdge(1, 4, 5), NewEdge(4, 5, 3), NewEdge(1, 6, 4)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 17)
	mates, weight, err = maxWeightMatching([]*Edge{NewEdge(1, 2, 9), NewEdge(1, 3, 8), NewEdge(2, 3, 10), NewEdge(1, 4, 5), NewEdge(4, 5, 3), NewEdge(3, 6, 4)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 6, 5, 4, 3}))
	is.True(weight == 16)
}

func TestMaxWeightMatching22(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 9), NewEdge(1, 3, 9), NewEdge(2, 3, 10), NewEdge(2, 4, 8), NewEdge(3, 5, 8), NewEdge(4, 5, 10), NewEdge(5, 6, 6)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 3, 4, 1, 2, 6, 5}))
	is.True(weight == 23)
}

func TestMaxWeightMatching23(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 10), NewEdge(1, 7, 10), NewEdge(2, 3, 12), NewEdge(3, 4, 20), NewEdge(3, 5, 20), NewEdge(4, 5, 25), NewEdge(5, 6, 10), NewEdge(6, 7, 10), NewEdge(7, 8, 8)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3, 6, 5, 8, 7}))
	is.True(weight == 48)
}

func TestMaxWeightMatching24(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 8), NewEdge(1, 3, 8), NewEdge(2, 3, 10), NewEdge(2, 4, 12), NewEdge(3, 5, 12), NewEdge(4, 5, 14), NewEdge(4, 6, 12), NewEdge(5, 7, 12), NewEdge(6, 7, 14), NewEdge(7, 8, 12)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 5, 6, 3, 4, 8, 7}))
	is.True(weight == 44)
}

func TestMaxWeightMatching25(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 23), NewEdge(1, 5, 22), NewEdge(1, 6, 15), NewEdge(2, 3, 25), NewEdge(3, 4, 22), NewEdge(4, 5, 25), NewEdge(4, 8, 14), NewEdge(5, 7, 13)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4}))
	is.True(weight == 67)
}

func TestMaxWeightMatching26(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 19), NewEdge(1, 3, 20), NewEdge(1, 8, 8), NewEdge(2, 3, 25), NewEdge(2, 4, 18), NewEdge(3, 5, 18), NewEdge(4, 5, 13), NewEdge(4, 7, 7), NewEdge(5, 6, 7)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 8, 3, 2, 7, 6, 5, 4, 1}))
	is.True(weight == 47)
}

func TestMaxWeightMatching30(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 45), NewEdge(1, 5, 45), NewEdge(2, 3, 50), NewEdge(3, 4, 45), NewEdge(4, 5, 50), NewEdge(1, 6, 30), NewEdge(3, 9, 35), NewEdge(4, 8, 35), NewEdge(5, 7, 26), NewEdge(9, 10, 5)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 146)
}

func TestMaxWeightMatching31(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 45), NewEdge(1, 5, 45), NewEdge(2, 3, 50), NewEdge(3, 4, 45), NewEdge(4, 5, 50), NewEdge(1, 6, 30), NewEdge(3, 9, 35), NewEdge(4, 8, 26), NewEdge(5, 7, 40), NewEdge(9, 10, 5)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 151)
}

func TestMaxWeightMatching32(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 45), NewEdge(1, 5, 45), NewEdge(2, 3, 50), NewEdge(3, 4, 45), NewEdge(4, 5, 50), NewEdge(1, 6, 30), NewEdge(3, 9, 35), NewEdge(4, 8, 28), NewEdge(5, 7, 26), NewEdge(9, 10, 5)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 139)
}

func TestMaxWeightMatching33(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 45), NewEdge(1, 7, 45), NewEdge(2, 3, 50), NewEdge(3, 4, 45), NewEdge(4, 5, 95), NewEdge(4, 6, 94), NewEdge(5, 6, 94), NewEdge(6, 7, 50), NewEdge(1, 8, 30), NewEdge(3, 11, 35), NewEdge(5, 9, 36), NewEdge(7, 10, 26), NewEdge(11, 12, 5)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 8, 3, 2, 6, 9, 4, 10, 1, 5, 7, 12, 11}))
	is.True(weight == 241)
}

func TestMaxWeightMatching34(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{NewEdge(1, 2, 40), NewEdge(1, 3, 40), NewEdge(2, 3, 60), NewEdge(2, 4, 55), NewEdge(3, 5, 55), NewEdge(4, 5, 50), NewEdge(1, 8, 15), NewEdge(5, 7, 30), NewEdge(7, 6, 10), NewEdge(8, 10, 10), NewEdge(4, 9, 30)}, false)
	is.NoErr(err)
	is.True(compareSlices(mates, []int{-1, 2, 1, 5, 9, 3, 7, 6, 10, 4, 8}))
	is.True(weight == 145)
}

func TestMaxWeightMatchingError(t *testing.T) {
	is := is.New(t)
	_, _, err := maxWeightMatching([]*Edge{NewEdge(-1, 2, 40), NewEdge(1, 3, 40)}, false)
	is.True(err != nil)
}
