package matching

import (
	"github.com/matryer/is"
	"testing"
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
	is.True(err == nil)
	is.True(compareSlices(mates, []int{}))
	is.True(weight == 0)
}

func TestMaxWeightMatching11(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{0, 1, 1}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{1, 0}))
	is.True(weight == 1)
}

func TestMaxWeightMatching12(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 10}, &Edge{2, 3, 11}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, -1, 3, 2}))
	is.True(weight == 11)
}

func TestMaxWeightMatching13(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 5}, &Edge{2, 3, 11}, &Edge{3, 4, 5}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, -1, 3, 2, -1}))
	is.True(weight == 11)
}

func TestMaxWeightMatching14(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 5}, &Edge{2, 3, 11}, &Edge{3, 4, 5}}, true)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3}))
	is.True(weight == 10)
}

func TestMaxWeightMatching16(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 2}, &Edge{1, 3, -2}, &Edge{2, 3, 1}, &Edge{2, 4, -1}, &Edge{3, 4, -6}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, -1, -1}))
	is.True(weight == 2)
	mates, weight, err = maxWeightMatching([]*Edge{&Edge{1, 2, 2}, &Edge{1, 3, -2}, &Edge{2, 3, 1}, &Edge{2, 4, -1}, &Edge{3, 4, -6}}, true)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 3, 4, 1, 2}))
	is.True(weight == -3)
}

func TestMaxWeightMatching20(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 8}, &Edge{1, 3, 9}, &Edge{2, 3, 10}, &Edge{3, 4, 7}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3}))
	is.True(weight == 15)
	mates, weight, err = maxWeightMatching([]*Edge{&Edge{1, 2, 8}, &Edge{1, 3, 9}, &Edge{2, 3, 10}, &Edge{3, 4, 7}, &Edge{1, 6, 5}, &Edge{4, 5, 6}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 21)
}

func TestMaxWeightMatching21(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 9}, &Edge{1, 3, 8}, &Edge{2, 3, 10}, &Edge{1, 4, 5}, &Edge{4, 5, 4}, &Edge{1, 6, 3}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 17)
	mates, weight, err = maxWeightMatching([]*Edge{&Edge{1, 2, 9}, &Edge{1, 3, 8}, &Edge{2, 3, 10}, &Edge{1, 4, 5}, &Edge{4, 5, 3}, &Edge{1, 6, 4}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 5, 4, 1}))
	is.True(weight == 17)
	mates, weight, err = maxWeightMatching([]*Edge{&Edge{1, 2, 9}, &Edge{1, 3, 8}, &Edge{2, 3, 10}, &Edge{1, 4, 5}, &Edge{4, 5, 3}, &Edge{3, 6, 4}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 6, 5, 4, 3}))
	is.True(weight == 16)
}

func TestMaxWeightMatching22(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 9}, &Edge{1, 3, 9}, &Edge{2, 3, 10}, &Edge{2, 4, 8}, &Edge{3, 5, 8}, &Edge{4, 5, 10}, &Edge{5, 6, 6}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 3, 4, 1, 2, 6, 5}))
	is.True(weight == 23)
}

func TestMaxWeightMatching23(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 10}, &Edge{1, 7, 10}, &Edge{2, 3, 12}, &Edge{3, 4, 20}, &Edge{3, 5, 20}, &Edge{4, 5, 25}, &Edge{5, 6, 10}, &Edge{6, 7, 10}, &Edge{7, 8, 8}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 4, 3, 6, 5, 8, 7}))
	is.True(weight == 48)
}

func TestMaxWeightMatching24(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 8}, &Edge{1, 3, 8}, &Edge{2, 3, 10}, &Edge{2, 4, 12}, &Edge{3, 5, 12}, &Edge{4, 5, 14}, &Edge{4, 6, 12}, &Edge{5, 7, 12}, &Edge{6, 7, 14}, &Edge{7, 8, 12}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 5, 6, 3, 4, 8, 7}))
	is.True(weight == 44)
}

func TestMaxWeightMatching25(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 23}, &Edge{1, 5, 22}, &Edge{1, 6, 15}, &Edge{2, 3, 25}, &Edge{3, 4, 22}, &Edge{4, 5, 25}, &Edge{4, 8, 14}, &Edge{5, 7, 13}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4}))
	is.True(weight == 67)
}

func TestMaxWeightMatching26(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 19}, &Edge{1, 3, 20}, &Edge{1, 8, 8}, &Edge{2, 3, 25}, &Edge{2, 4, 18}, &Edge{3, 5, 18}, &Edge{4, 5, 13}, &Edge{4, 7, 7}, &Edge{5, 6, 7}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 8, 3, 2, 7, 6, 5, 4, 1}))
	is.True(weight == 47)
}

func TestMaxWeightMatching30(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 45}, &Edge{1, 5, 45}, &Edge{2, 3, 50}, &Edge{3, 4, 45}, &Edge{4, 5, 50}, &Edge{1, 6, 30}, &Edge{3, 9, 35}, &Edge{4, 8, 35}, &Edge{5, 7, 26}, &Edge{9, 10, 5}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 146)
}

func TestMaxWeightMatching31(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 45}, &Edge{1, 5, 45}, &Edge{2, 3, 50}, &Edge{3, 4, 45}, &Edge{4, 5, 50}, &Edge{1, 6, 30}, &Edge{3, 9, 35}, &Edge{4, 8, 26}, &Edge{5, 7, 40}, &Edge{9, 10, 5}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 151)
}

func TestMaxWeightMatching32(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 45}, &Edge{1, 5, 45}, &Edge{2, 3, 50}, &Edge{3, 4, 45}, &Edge{4, 5, 50}, &Edge{1, 6, 30}, &Edge{3, 9, 35}, &Edge{4, 8, 28}, &Edge{5, 7, 26}, &Edge{9, 10, 5}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 6, 3, 2, 8, 7, 1, 5, 4, 10, 9}))
	is.True(weight == 139)
}

func TestMaxWeightMatching33(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 45}, &Edge{1, 7, 45}, &Edge{2, 3, 50}, &Edge{3, 4, 45}, &Edge{4, 5, 95}, &Edge{4, 6, 94}, &Edge{5, 6, 94}, &Edge{6, 7, 50}, &Edge{1, 8, 30}, &Edge{3, 11, 35}, &Edge{5, 9, 36}, &Edge{7, 10, 26}, &Edge{11, 12, 5}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 8, 3, 2, 6, 9, 4, 10, 1, 5, 7, 12, 11}))
	is.True(weight == 241)
}

func TestMaxWeightMatching34(t *testing.T) {
	is := is.New(t)
	mates, weight, err := maxWeightMatching([]*Edge{&Edge{1, 2, 40}, &Edge{1, 3, 40}, &Edge{2, 3, 60}, &Edge{2, 4, 55}, &Edge{3, 5, 55}, &Edge{4, 5, 50}, &Edge{1, 8, 15}, &Edge{5, 7, 30}, &Edge{7, 6, 10}, &Edge{8, 10, 10}, &Edge{4, 9, 30}}, false)
	is.True(err == nil)
	is.True(compareSlices(mates, []int{-1, 2, 1, 5, 9, 3, 7, 6, 10, 4, 8}))
	is.True(weight == 145)
}

func TestMaxWeightMatchingError(t *testing.T) {
	is := is.New(t)
	_, _, err := maxWeightMatching([]*Edge{&Edge{-1, 2, 40}, &Edge{1, 3, 40}}, false)
	is.True(err != nil)
}
