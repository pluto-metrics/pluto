package query

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/pluto-metrics/pluto/pkg/config"
)

const (
	headerClickhouseQueryID = "X-ClickHouse-Query-Id"
	headerUserAgent         = "User-Agent"
	clientName              = "pluto/v0.1.0"
	defaultUser             = "default"
)

// Opts ...
type Opts struct {
	BufferSize int
	HTTPClient *http.Client
	QueryID    string
	Headers    map[string]string
	Discovery  func(ctx context.Context, dsn string) (string, error)
}

// Request ...
type Request struct {
	sync.Mutex
	ctx       context.Context
	ctxCancel context.CancelFunc
	reader    *io.PipeReader
	writer    *io.PipeWriter
	writerBuf *bufio.Writer

	finished chan interface{}
	respErr  error     // read/write with mutex
	resp     *Response // read/write with mutex

	vars struct {
		reqBodyBytes int
	}
}

// Response ...
type Response struct {
	ctx      context.Context
	httpBody io.ReadCloser

	vars struct {
		respBodyBytes int
	}
}

// New starts a new request to ClickHouse
func NewRequest(ctx context.Context, cfg config.ClickHouse, opts Opts) (*Request, error) {
	var err error
	dsn := cfg.DSN
	if opts.Discovery != nil {
		dsn, err = opts.Discovery(ctx, dsn)
		if err != nil {
			return nil, err
		}
	}

	u, err := url.Parse(cfg.DSN)
	if err != nil {
		return nil, err
	}

	// 1mb minimum size of buffer
	if opts.BufferSize < 1024*1024 {
		opts.BufferSize = 1024 * 1024
	}

	reader, writer := io.Pipe()

	writerBuf := bufio.NewWriterSize(writer, opts.BufferSize)

	ctx, ctxCancel := context.WithCancel(ctx)

	req := &Request{
		reader:    reader,
		writer:    writer,
		writerBuf: writerBuf,
		ctx:       ctx,
		ctxCancel: ctxCancel,
		finished:  make(chan interface{}),
	}

	url := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}
	values := url.Query()
	headers := http.Header{}
	headers.Set(headerUserAgent, clientName)

	uv := u.Query()
	// params from dsn
	for k := range uv {
		if strings.HasPrefix(strings.ToLower(k), "x-clickhouse-") {
			headers.Set(k, uv.Get(k))
		} else {
			values.Set(k, uv.Get(k))
		}
	}

	// params from separate config
	for k, v := range cfg.Params {
		if strings.HasPrefix(strings.ToLower(k), "x-clickhouse-") {
			headers.Set(k, v)
		} else {
			values.Set(k, v)
		}
	}

	if len(opts.QueryID) > 0 {
		headers.Set(headerClickhouseQueryID, opts.QueryID)
	}

	for k, v := range opts.Headers {
		headers.Set(k, v)
	}

	url.RawQuery = values.Encode()

	httpReq := (&http.Request{
		Method:           "POST",
		ProtoMajor:       1,
		ProtoMinor:       1,
		URL:              url,
		TransferEncoding: []string{"chunked"},
		Body:             req.reader,
		Header:           headers,
	}).WithContext(ctx)

	if u.User.Username() != "" {
		password, _ := u.User.Password()
		httpReq.SetBasicAuth(u.User.Username(), password)
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	go func() {
		httpResp, err := httpClient.Do(httpReq)
		defer close(req.finished)

		if err != nil {
			req.Lock()
			req.respErr = err
			req.Unlock()
			return
		}

		if httpResp.StatusCode != http.StatusOK {
			body, bodyErr := io.ReadAll(httpResp.Body)

			if bodyErr != nil {
				err = fmt.Errorf("http status code %d, but can't read response body: %s", httpResp.StatusCode, bodyErr.Error())
			} else {
				err = fmt.Errorf("http status code %d: %s", httpResp.StatusCode, string(body))
			}
			req.Lock()
			req.respErr = err
			req.Unlock()
			return
		}

		req.Lock()
		req.resp = &Response{ctx: ctx, httpBody: httpResp.Body}
		req.Unlock()
	}()

	return req, nil
}

// Write ...
func (req *Request) Write(p []byte) (int, error) {
	n, err := req.writerBuf.Write(p)
	req.vars.reqBodyBytes += n
	return n, err
}

func (req *Request) WriteByte(b byte) error {
	req.vars.reqBodyBytes += 1
	return req.writerBuf.WriteByte(b)
}

// Close ...
func (req *Request) Close() error {
	req.ctxCancel()

	<-req.finished

	req.Lock()
	err := req.respErr
	resp := req.resp
	req.Unlock()

	if err != nil {
		return err
	}

	if resp != nil {
		return resp.Close()
	}

	return nil
}

// Finish finishes the request and starts reading the response
func (req *Request) Finish() (*Response, error) {

	if err := req.writerBuf.Flush(); err != nil {
		// may be an error from the server
		req.Lock()
		respErr := req.respErr
		req.Unlock()
		if respErr != nil {
			return nil, errors.WithStack(respErr)
		}
		return nil, errors.WithStack(err)
	}
	if err := req.writer.Close(); err != nil {
		// may be an error from the server
		req.Lock()
		respErr := req.respErr
		req.Unlock()
		if respErr != nil {
			return nil, errors.WithStack(respErr)
		}
		return nil, errors.WithStack(err)
	}

	<-req.finished

	req.Lock()
	err := req.respErr
	resp := req.resp
	req.Unlock()

	return resp, err
}

// Read reads data from the response
func (resp *Response) Read(p []byte) (int, error) {
	if resp == nil {
		return 0, fmt.Errorf("response is nil")
	}
	n, err := resp.httpBody.Read(p)

	resp.vars.respBodyBytes += n

	return n, err
}

// Close closes the response and discards any remaining body
func (resp *Response) Close() error {
	_, err := io.Copy(io.Discard, resp)
	return err
}
