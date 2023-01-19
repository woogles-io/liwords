package runemapping

import (
	"reflect"
	"testing"
)

func TestUserVisible(t *testing.T) {
	// Initialize an alphabet.
	rm := &RuneMapping{}
	rm.Init()
	rm.Update("AEROLITH")
	rm.Update("HOMEMADE")
	rm.Update("GAMODEME")
	rm.Update("XU")
	rm.Reconcile(nil)
	expected := LetterSlice([]rune{
		'A', 'D', 'E', 'G', 'H', 'I', 'L', 'M', 'O', 'R', 'T', 'U', 'X'})
	if !reflect.DeepEqual(rm.letterSlice, expected) {
		t.Errorf("Did not equal, expected %v got %v", expected, rm.letterSlice)
	}
	mw := MachineWord([]MachineLetter{5, 9, 8, 6, 3})
	uv := mw.UserVisible(rm)
	if uv != "HOMIE" {
		t.Errorf("Did not equal, expected %v got %v", "HOMIE", uv)
	}

	mw2 := MachineWord([]MachineLetter{13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0})
	uv2 := mw2.UserVisible(rm)
	if uv2 != "XUTROMLIHGEDA?" {
		t.Errorf("Did not equal, expected %v got %v", "XUTROMLIHGEDA?", uv2)
	}
}

func TestUserVisibleWithBlank(t *testing.T) {
	// Initialize an alphabet.
	rm := &RuneMapping{}
	rm.Init()
	rm.Update("AEROLITH")
	rm.Update("HOMEMADE")
	rm.Update("GAMODEME")
	rm.Update("XU")
	rm.Reconcile(nil)

	mw := MachineWord([]MachineLetter{251, 247, 248, 250, 253})
	uv := mw.UserVisible(rm)
	if uv != "homie" {
		t.Errorf("Did not equal, expected %v got %v", "homie", uv)
	}
}
