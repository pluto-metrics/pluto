package listen

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type HTTP struct {
	sync.Mutex
	mp map[string]*http.ServeMux
}

func NewHTTP() *HTTP {
	return &HTTP{
		mp: make(map[string]*http.ServeMux),
	}
}

func (h *HTTP) Mux(addr string) *http.ServeMux {
	h.Lock()
	defer h.Unlock()

	mux, exists := h.mp[addr]
	if exists {
		return mux
	}

	mux = http.NewServeMux()
	h.mp[addr] = mux
	return mux
}

func (h *HTTP) Run(ctx context.Context) error {
	h.Lock()
	defer h.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)

	for addr, mux := range h.mp {
		httpSrv := &http.Server{
			Addr:        addr,
			Handler:     mux,
			ReadTimeout: 10 * time.Second,
		}

		go func() {
			err := httpSrv.ListenAndServe()
			errChan <- err
		}()

		go func() {
			<-ctx.Done()
			httpSrv.Close()
		}()
	}

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return errors.New("context cancelled")
	}
}
