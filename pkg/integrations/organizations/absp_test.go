package organizations

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

// lines of absp-database.org/current.txt as it is laid out, trailing spaces
// and all: a blank MEMNO leaves its two spaces, so the GAMES after it (15, 28)
// is not a membership number
const abspRatingsFixture = "ELDAR(GM) DAVID 205 1 02279 \n" +
	"TEITELBAUM JAKOB 204 2  15\n" +
	"RICHARDS(GM) NIGEL 201 3  \n" +
	"TACHIE-MENSON CHARLES 193 12  28\n" +
	"BALOGUN RASHEED 185 23 02524 111\n" +
	"COHEN(EXP) EVAN 183 28 02333 \n" +
	"O'ROURKE NUALA 154 238 00408 \n" +
	"DE_POITIERS WANDA 132 484 00732 \n" +
	"THOMPSON PAUL{HALIFAX} 169 102 01649 \n" +
	"STEVENS  MARY_ALINE 131 499  16\n" +
	"PUFFER SHEILA  51 1381  2\n" +
	"SIMMONS(GM) ALLAN 195 7 00007 \n" +
	"YI_AN TAY 147 323  7\n"

func TestABSPMemberID(t *testing.T) {
	is := is.New(t)
	for in, want := range map[string]string{
		"00745": "745",
		"745":   "745",
		"00000": "0",
		"02279": "2279",
	} {
		got, ok := abspMemberID(in)
		is.True(ok)
		is.Equal(got, want)
	}
	for _, in := range []string{"", "GM", "007a", "-5", "1 2"} {
		_, ok := abspMemberID(in)
		is.True(!ok) // not a membership number
	}
}

func TestParseABSPRatings(t *testing.T) {
	is := is.New(t)
	players, err := parseABSPRatings([]byte(abspRatingsFixture))
	is.NoErr(err)

	// only members are kept, by their number without its leading zeros
	is.Equal(players, map[string]abspPlayer{
		"2279": {MemberID: "2279", Name: "David Eldar", Title: "GM"},
		"2524": {MemberID: "2524", Name: "Rasheed Balogun", Title: ""},
		"2333": {MemberID: "2333", Name: "Evan Cohen", Title: "EXP"},
		"408":  {MemberID: "408", Name: "Nuala O'Rourke", Title: ""},
		"732":  {MemberID: "732", Name: "Wanda De Poitiers", Title: ""},
		"1649": {MemberID: "1649", Name: "Paul Thompson", Title: ""},
		"7":    {MemberID: "7", Name: "Allan Simmons", Title: "GM"},
	})

	for _, body := range []string{
		``,
		"\n\n",
		// nonmembers only
		"TEITELBAUM JAKOB 204 2  15\n",
		// not the ratings list
		"<!DOCTYPE html>\n<html></html>\n",
		"ELDAR(GM) DAVID 205 1 02279 \nELDAR DAVID\n",
		// a title it does not know
		"ELDAR(IM) DAVID 205 1 02279 \n",
		// a membership number twice
		"ELDAR(GM) DAVID 205 1 02279 \nELDAR DAVE 205 2 02279 \n",
		// a membership number that is not five digits
		"ELDAR(GM) DAVID 205 1 2279 \n",
	} {
		_, err := parseABSPRatings([]byte(body))
		is.True(err != nil) // a broken list is refused
	}
}

func abspServer(t *testing.T, ratings *string, status *int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/current.txt":
			w.WriteHeader(*status)
			w.Write([]byte(*ratings))
		case "/absp_log_on_or_off.php":
			if r.FormValue("action") == "password_logon" &&
				r.FormValue("username") == "evan@example.com" && r.FormValue("password") == "right" {
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "evan"})
				w.Write([]byte("<p style='color:green'>ABSP Member</p>"))
				return
			}
			w.Write([]byte("<!-- absp_log_on_or_off.php -->"))
		case "/edit_absp_member.php":
			if c, err := r.Cookie("PHPSESSID"); err != nil || c.Value != "evan" {
				w.Write([]byte("<html>log in first</html>"))
				return
			}
			w.Write([]byte("<table><tr><td>absp_number:</td>\n<td>\n02333</td></tr>\n" +
				"<tr><td>first_name:</td>\n<td>\nEvan</td></tr>\n" +
				"<tr><td>last_name:</td>\n<td>\nCohen</td></tr></table>"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestABSPFetchTitle(t *testing.T) {
	is := is.New(t)
	ratings, status := abspRatingsFixture, http.StatusOK
	server := abspServer(t, &ratings, &status)
	absp := NewABSPIntegration()
	absp.BaseURL = server.URL

	// with no credentials, the list is looked up as it is
	info, err := absp.FetchTitle("02279", nil)
	is.NoErr(err)
	is.Equal(info.MemberID, "2279")
	is.Equal(info.FullName, "David Eldar")
	is.Equal(info.RawTitle, "GM")
	is.Equal(info.Organization, OrgABSP)

	// a member listed with no title holds none
	info, err = absp.FetchTitle("2524", nil)
	is.NoErr(err)
	is.Equal(info.RawTitle, "")
	is.Equal(info.FullName, "Rasheed Balogun")

	// a member missing from the list, or a nonmember's GAMES column, comes
	// back with no title and no name
	for _, id := range []string{"15", "16", "999"} {
		info, err = absp.FetchTitle(id, nil)
		is.NoErr(err)
		is.Equal(info.RawTitle, "")
		is.Equal(info.FullName, "")
		_, err = absp.GetRealName(id, nil)
		is.True(err != nil) // only a listed member has a name
	}
	// member 7 is Allan Simmons, not the nonmember with 7 games
	info, err = absp.FetchTitle("00007", nil)
	is.NoErr(err)
	is.Equal(info.FullName, "Allan Simmons")
	is.Equal(info.RawTitle, "GM")

	_, err = absp.FetchTitle("EVAN", nil)
	is.True(err != nil) // not a membership number

	// with credentials, the profile's number and name are used
	creds := map[string]string{"username": "evan@example.com", "password": "right"}
	info, err = absp.FetchTitle("", creds)
	is.NoErr(err)
	is.Equal(info.MemberID, "2333")
	is.Equal(info.FullName, "Evan Cohen")
	is.Equal(info.RawTitle, "EXP")
	name, err := absp.GetRealName("", creds)
	is.NoErr(err)
	is.Equal(name, "Evan Cohen")

	_, err = absp.FetchTitle("", map[string]string{"username": "evan@example.com", "password": "wrong"})
	is.True(err != nil) // a wrong password
	_, err = absp.FetchTitle("", map[string]string{"username": "evan@example.com"})
	is.True(err != nil) // no password

	// the list is downloaded once per instance; a failure is kept too
	failing := NewABSPIntegration()
	failing.BaseURL = server.URL
	status = http.StatusInternalServerError
	_, err = failing.FetchTitle("2279", nil)
	is.True(err != nil)
	status = http.StatusOK
	_, err = failing.FetchTitle("2279", nil)
	is.True(err != nil)
}

func TestABSPName(t *testing.T) {
	is := is.New(t)
	for in, want := range map[string]string{
		"ELDAR":           "Eldar",
		"O'ROURKE":        "O'Rourke",
		"TACHIE-MENSON":   "Tachie-Menson",
		"DE_POITIERS":     "De Poitiers",
		"KONG_CHOCK":      "Kong Chock",
		"JENNER-O'SHEA":   "Jenner-O'Shea",
		"MUZAN_EKPELU":    "Muzan Ekpelu",
		"LAMABADUSURIYA":  "Lamabadusuriya",
		"TSANG_WAI_YIN":   "Tsang Wai Yin",
		"RUSSELL-GEBBETT": "Russell-Gebbett",
	} {
		is.Equal(abspName(in), want)
	}
}
