package main

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// teeBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
// It returns an error if the initial slurp of all bytes fails.
func teeBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

type ResponseRecorder struct {
	rw     http.ResponseWriter
	status int
	writer io.Writer
	header map[string][]string
}

func (rec *ResponseRecorder) WriteHeader(code int) {
	rec.status = code
	rec.rw.WriteHeader(code)
}

func (rec *ResponseRecorder) Write(b []byte) (int, error) {
	return rec.writer.Write(b)
}

func (rec *ResponseRecorder) Header() http.Header {
	return rec.rw.Header()
}

// Response Recorder
func responseRecorder(rw http.ResponseWriter, status int, writer io.Writer) ResponseRecorder {
	rr := ResponseRecorder{
		rw,
		status,
		writer,
		make(map[string][]string, 5),
	}
	return rr
}

func ErrorReqResLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
		var buf bytes.Buffer
		multiWriter := io.MultiWriter(rw, &buf)

		// Initialize the status to 200 in case WriteHeader is not called
		response := responseRecorder(
			rw,
			200,
			multiWriter,
		)
		requestTime := time.Now().UTC()
		var body1, body2 io.ReadCloser
		var err error
		// buffer the entire request body into memory for logging
		if body1, body2, err = teeBody(request.Body); err != nil {
			log.Printf("Error while reading request body: %v.\n", err)
		} else {
			// Body is a ReadCloser meaning that it does not implement the Seek interface
			// It must be buffered into memory to be read more than once
			// This is a replacement reader reading the buffer for the original server handler
			request.Body = body1
		}

		next.ServeHTTP(&response, request)

		// Response Time
		responseTime := time.Now().UTC()

		// this is a separate ReadCloser, reading the same buffer as above for logging
		request.Body = body2

		logEvent(request, response, buf.Bytes(), requestTime, responseTime)

	})
}

func logEvent(req *http.Request, response ResponseRecorder, rspBuf []byte, reqTime time.Time, respTime time.Time) {
	if response.status == 200 {
		return
	}
	logEvt := log.Error()
	logEvt = logEvt.Str("from", req.RemoteAddr)

	if response.Header().Get("content-type") == "application/protobuf" {
		logEvt = logEvt.Bytes("resp-buffer", rspBuf)
	} else {
		logEvt = logEvt.Str("resp", string(rspBuf))
	}

	readReqBody, reqBodyErr := io.ReadAll(req.Body)
	if reqBodyErr != nil {
		log.Info().Msg("error-reading-req-body")
	}
	if len(readReqBody) > 0 {
		if req.Header.Get("content-type") == "application/protobuf" {
			logEvt = logEvt.Bytes("req-buffer", readReqBody)
		} else {
			logEvt = logEvt.Str("req", string(readReqBody))
		}
	}

	logEvt.Str("method", req.Method).
		Str("path", req.URL.Path).
		Interface("headers", req.Header).
		Time("reqTime", reqTime).
		Time("respTime", respTime).
		Msg("non-ok-status")
}
