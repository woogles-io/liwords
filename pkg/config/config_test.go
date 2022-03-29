package config

import (
	"testing"

	"github.com/matryer/is"
)

func TestPGConnStringToUrl(t *testing.T) {
	is := is.New(t)
	cs := "host=db port=5432 user=postgres dbname=liwords password=pass sslmode=disable"

	is.Equal(PGConnStringToUrl(cs), "postgres://postgres:pass@db:5432/liwords?sslmode=disable")
}
