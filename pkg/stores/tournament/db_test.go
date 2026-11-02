package tournament

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	tl "github.com/woogles-io/liwords/pkg/tournament"
)

// The divisions column as it was decoded and encoded before storedDivision:
// each division's manager held as raw JSON, then decoded in a second pass.
func decodeDivisionsTwoPass(t *testing.T, raw []byte) map[string]*entity.TournamentDivision {
	t.Helper()
	var divisions map[string]*entity.TournamentDivision
	if err := json.Unmarshal(raw, &divisions); err != nil {
		t.Fatal(err)
	}
	for _, division := range divisions {
		var classic tl.ClassicDivision
		if err := json.Unmarshal(division.DivisionRawMessage, &classic); err != nil {
			t.Fatal(err)
		}
		division.DivisionManager = &classic
		division.DivisionRawMessage = nil
	}
	return divisions
}

func encodeDivisionsTwoPass(t *testing.T, divisions map[string]*entity.TournamentDivision) []byte {
	t.Helper()
	for _, division := range divisions {
		dmJSON, err := json.Marshal(division.DivisionManager)
		if err != nil {
			t.Fatal(err)
		}
		division.DivisionRawMessage = dmJSON
	}
	out, err := json.Marshal(divisions)
	if err != nil {
		t.Fatal(err)
	}
	for _, division := range divisions {
		division.DivisionRawMessage = nil
	}
	return out
}

// Decoding and encoding the divisions in one pass must read and write exactly
// what the two-pass code did, in every combination, so rows written by either
// version load in the other.
func TestDivisionsSinglePassMatchesTwoPass(t *testing.T) {
	is := is.New(t)
	raw, err := os.ReadFile("../../tournament/testdata/wtf5.json")
	is.NoErr(err)
	s := &DBStore{}
	decode := func() map[string]*entity.TournamentDivision {
		tm, err := s.dbObjToEntity(&tournament{Divisions: raw, Directors: []byte("{}")})
		is.NoErr(err)
		return tm.Divisions
	}
	encode := func(divisions map[string]*entity.TournamentDivision) []byte {
		dbt, err := s.toDBObj(&entity.Tournament{Divisions: divisions})
		is.NoErr(err)
		return dbt.Divisions
	}

	want := encodeDivisionsTwoPass(t, decodeDivisionsTwoPass(t, raw))
	is.True(len(want) > 1000)

	is.Equal(string(encode(decode())), string(want))
	is.Equal(string(encodeDivisionsTwoPass(t, decode())), string(want))
	is.Equal(string(encode(decodeDivisionsTwoPass(t, raw))), string(want))
}

// Encoding must not leave the encoded JSON on the entity. The old code stored
// it in DivisionRawMessage, where a cached tournament kept a copy of every
// division's JSON alive until the next save replaced it.
func TestEncodeLeavesNoRawJSONOnEntity(t *testing.T) {
	is := is.New(t)
	raw, err := os.ReadFile("../../tournament/testdata/wtf5.json")
	is.NoErr(err)
	s := &DBStore{}
	tm, err := s.dbObjToEntity(&tournament{Divisions: raw, Directors: []byte("{}")})
	is.NoErr(err)
	_, err = s.toDBObj(tm)
	is.NoErr(err)
	for _, division := range tm.Divisions {
		is.Equal(len(division.DivisionRawMessage), 0)
	}
}

func TestDecodeRejectsUnknownManagerType(t *testing.T) {
	is := is.New(t)
	s := &DBStore{}
	_, err := s.dbObjToEntity(&tournament{
		Divisions: []byte(`{"A": {"mgrType": 7, "json": {}}}`),
		Directors: []byte("{}"),
	})
	is.True(err != nil)
}
