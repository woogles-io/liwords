package gameplay

import (
	"context"

	"github.com/domino14/crosswords/pkg/entity"
)

type playerstore interface {
	Get(ctx context.Context, username string) (*entity.Player, error)
	Set(context.Context, *entity.Player) error
}
