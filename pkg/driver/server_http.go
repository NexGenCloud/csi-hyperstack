package driver

import (
	"context"
	"fmt"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"sync"
)

func RunHttpServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	endpoint string,
	metricsEnabled bool,
) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		_, _ = fmt.Fprintf(w, "ok")
	})

	if metricsEnabled {
		klog.Infof("metrics available in %s/metrics", endpoint)
		mux.Handle("/metrics", legacyregistry.HandlerWithReset())
	}

	srv := &http.Server{
		BaseContext: func(net.Listener) context.Context { return ctx },
		Addr:        endpoint,
		Handler:     mux,
	}

	go func() {
		defer wg.Done()

		klog.Infof("Running server on %q", endpoint)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			klog.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv
}
