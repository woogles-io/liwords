package tournament

import (
	"testing"

	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

func TestValidateSlugVisitable(t *testing.T) {
	cases := []struct {
		slug    string
		wantErr bool
	}{
		{"/tournament/foo", false},
		{"/tournament/foo-bar_1.2", false},
		{"/tournament/test 1", true}, // embedded space
		{"/tournament/trailing ", true},
		{"/tournament/a%20b", true}, // "%" can smuggle in a space
		{"/tournament/a+b", true},   // "+" can mean a space
		{"/tournament/who?x", true}, // query delimiter
		{"/tournament/a<b", true},   // a browser would percent-encode this
	}
	for _, c := range cases {
		_, err := validateTournamentTypeMatchesSlug(pb.TType_STANDARD, c.slug)
		if (err != nil) != c.wantErr {
			t.Errorf("validateTournamentTypeMatchesSlug(%q) err=%v, wantErr=%v",
				c.slug, err, c.wantErr)
		}
	}
}
