package organizations

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

// laid out as the published list is
const sgTitlesFixture = `{"players": [
  {"id": "4", "name": "Law, Sarah", "title": "M"}
, {"id": "5", "name": "Sim, Tony", "title": "GM"}
, {"id": "89", "name": "Gan Cher Siong", "title": "NM"}
, {"id": "90", "name": "Chung, Eddie", "title": "SM"}
]}
`

func TestSGMemberID(t *testing.T) {
	is := is.New(t)
	for in, want := range map[string]string{
		"0":                                 "0",
		"00000":                             "0",
		"01234":                             "1234",
		"1234":                              "1234",
		"000123456789012345678901234567890": "123456789012345678901234567890",
	} {
		got, ok := sgMemberID(in)
		is.True(ok)
		is.Equal(got, want)
	}
	for _, in := range []string{"", "M", "01234M", "+5", "-5", " 1234", "1234 ", "12 34", "1\n"} {
		_, ok := sgMemberID(in)
		is.True(!ok) // not a membership number
	}
}

func TestParseSGTitles(t *testing.T) {
	is := is.New(t)
	players, err := parseSGTitles([]byte(sgTitlesFixture))
	is.NoErr(err)
	is.Equal(len(players), 4)
	is.Equal(players["89"], sgPlayer{ID: "89", Name: "Gan Cher Siong", Title: "NM"})

	// ids are kept in their canonical form
	players, err = parseSGTitles([]byte(`{"players": [{"id": "0042", "name": "A", "title": "M"}]}`))
	is.NoErr(err)
	is.Equal(players["42"].ID, "42")

	// a member listed with an empty title holds none
	players, err = parseSGTitles([]byte(`{"players": [{"id": "4", "name": "A", "title": ""}]}`))
	is.NoErr(err)
	is.Equal(players["4"], sgPlayer{ID: "4", Name: "A", Title: ""})

	// an empty list is valid: nobody holds a title
	players, err = parseSGTitles([]byte(`{"players": []}`))
	is.NoErr(err)
	is.Equal(len(players), 0)

	for _, body := range []string{
		``,
		`{"players": [`,
		`{"players": [{"id": "4", "name": "A", "title": "M"}`,
		`{"players": null}`,
		`{}`,
		`[]`,
		`{"players": [{"id": "4", "name": "A", "title": "M"}]} {}`,
		`{"players": [{"id": "4", "name": "A", "title": "M"}, {"id": "04", "name": "B", "title": "GM"}]}`,
		`{"players": [{"id": "4M", "name": "A", "title": "M"}]}`,
		`{"players": [{"id": "", "name": "A", "title": "M"}]}`,
		`{"players": [{"id": 4, "name": "A", "title": "M"}]}`,
		`{"players": [{"id": "4", "name": "A", "title": "IM"}]}`,
		`{"players": [{"id": "4", "name": "A", "title": "gm"}]}`,
		`{"players": [{"id": "4", "name": "A", "title": null}]}`,
		`{"players": [{"id": "4", "name": "A"}]}`,
		`{"players": [{"id": "4", "name": "", "title": "M"}]}`,
		`{"players": [{"id": "4", "title": "M"}]}`,
	} {
		_, err := parseSGTitles([]byte(body))
		if err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

func TestSGFetchTitle(t *testing.T) {
	is := is.New(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sgTitlesFixture))
	}))
	defer server.Close()

	sg := NewSGIntegration()
	sg.TitlesURL = server.URL

	info, err := sg.FetchTitle("0089", nil)
	is.NoErr(err)
	is.Equal(info.Organization, OrgSG)
	is.Equal(info.MemberID, "89")
	is.Equal(info.RawTitle, "NM")
	is.Equal(info.FullName, "Gan Cher Siong")

	// a member missing from the list holds no title, and is not an error
	info, err = sg.FetchTitle("7", nil)
	is.NoErr(err)
	is.Equal(info.MemberID, "7")
	is.Equal(info.RawTitle, "")
	is.Equal(info.FullName, "")

	name, err := sg.GetRealName("5", nil)
	is.NoErr(err)
	is.Equal(name, "Sim, Tony")

	// but only a listed member can be linked
	_, err = sg.GetRealName("7", nil)
	is.True(err != nil)

	_, err = sg.FetchTitle("5M", nil)
	is.True(err != nil)

	is.Equal(requests, 1) // one download per instance
}

func TestSGFetchTitleUntitledAndEmpty(t *testing.T) {
	is := is.New(t)
	serve := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(body))
		}))
	}

	// a member listed with no title holds none, and can be linked
	server := serve(`{"players": [{"id": "4", "name": "Law, Sarah", "title": ""}]}`)
	sg := NewSGIntegration()
	sg.TitlesURL = server.URL
	info, err := sg.FetchTitle("4", nil)
	is.NoErr(err)
	is.Equal(info.RawTitle, "")
	is.Equal(info.FullName, "Law, Sarah")
	name, err := sg.GetRealName("4", nil)
	is.NoErr(err)
	is.Equal(name, "Law, Sarah")
	server.Close()

	// an empty list takes every title away, and links no one
	server = serve(`{"players": []}`)
	sg = NewSGIntegration()
	sg.TitlesURL = server.URL
	info, err = sg.FetchTitle("5", nil)
	is.NoErr(err)
	is.Equal(info.RawTitle, "")
	is.Equal(info.FullName, "")
	_, err = sg.GetRealName("5", nil)
	is.True(err != nil)
	server.Close()
}

func TestSGFetchTitleFailures(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		status int
		body   string
	}{
		{http.StatusNotFound, sgTitlesFixture},
		{http.StatusOK, `{"players": null}`},
		{http.StatusOK, `{"players": [`},
	} {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.WriteHeader(tc.status)
			w.Write([]byte(tc.body))
		}))
		sg := NewSGIntegration()
		sg.TitlesURL = server.URL
		_, err := sg.FetchTitle("5", nil)
		is.True(err != nil) // a failed download changes no one's title
		_, err = sg.GetRealName("5", nil)
		is.True(err != nil)
		is.Equal(requests, 1) // and is not downloaded again by the instance
		server.Close()
	}
}
