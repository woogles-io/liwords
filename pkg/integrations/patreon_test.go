package integrations

import (
	"testing"

	"github.com/matryer/is"
)

func TestHighestTier(t *testing.T) {
	is := is.New(t)
	pm := &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"22998862", "tier"}, {"24128408", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierGoldenRetriever)

	pm = &PatreonMemberDatum{}
	is.Equal(HighestTier(pm), TierNone)

	pm = &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"22998862", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierChihuahua)

	pm = &PatreonMemberDatum{
		Relationships: PatreonRelationships{
			CurrentlyEntitledTiers: EntitledTiersRelationship{
				Data: []PatreonRelationship{
					{"24128312", "tier"}, {"10805942", "tier"},
				},
			},
		},
	}
	is.Equal(HighestTier(pm), TierDalmatian)

}
