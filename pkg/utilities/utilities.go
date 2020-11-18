package utilities

import (
	"fmt"
	"strings"
)

func MinArr(array []int) int {
	if len(array) == 0 {
		return 0
	}
	minInt := array[0]
	for _, j := range array {
		if j < minInt {
			minInt = j
		}
	}
	return minInt
}

func UminArr(array []uint64) uint64 {
	if len(array) == 0 {
		return 0
	}
	minInt := array[0]
	for _, j := range array {
		if j < minInt {
			minInt = j
		}
	}
	return minInt
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func Umax(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func IndexOf(v int, array *[]int) int {
	for i, j := range *array {
		if v == j {
			return i
		}
	}
	return -1
}

func Reverse(array []int) {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}
}

func IntArrayToString(array []int) string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(array)), ", "), "[]")
}
