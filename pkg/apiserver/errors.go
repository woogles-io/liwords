package apiserver

import (
	"errors"

	"connectrpc.com/connect"
)

func Unauthenticated(str string) *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated, errors.New(str))
}

func PermissionDenied(str string) *connect.Error {
	return connect.NewError(connect.CodePermissionDenied, errors.New(str))
}

func InvalidArg(str string) *connect.Error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(str))
}

// InternalErr sends a 500, it is a good way of transparently passing through an error.
func InternalErr(err error) *connect.Error {
	return connect.NewError(connect.CodeInternal, err)
}

func NotFound(str string) *connect.Error {
	return connect.NewError(connect.CodeNotFound, errors.New(str))
}
