package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/disposableemail"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

func TestDisposableEmailCheck(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mailinator.com\n"))
	}))
	defer srv.Close()
	cs := &ConfigService{disposable: disposableemail.NewList(srv.URL)}
	check := cs.disposableEmailCheck(context.Background())

	is.Equal(check("someone@mailinator.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_YES)
	is.Equal(check("someone@gmail.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_NO)
}

func TestDisposableEmailCheckWithoutTheListIsUnknown(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	for _, cs := range []*ConfigService{
		{disposable: disposableemail.NewList(srv.URL)},
		{},
	} {
		check := cs.disposableEmailCheck(context.Background())
		is.Equal(check("someone@mailinator.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_UNKNOWN)
		is.Equal(check("someone@gmail.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_UNKNOWN)
	}
}
