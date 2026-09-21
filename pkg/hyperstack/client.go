package hyperstack

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

type HyperstackClient struct {
	Client    *http.Client
	ApiKey    string
	ApiServer string
	Version   string
}

func NewHyperstackClient(
	apiKey string,
	apiServer string,
	version string,
) *HyperstackClient {
	return &HyperstackClient{
		Client:    http.DefaultClient,
		ApiKey:    apiKey,
		ApiServer: apiServer,
		Version:   version,
	}
}

func (c HyperstackClient) GetAddHeadersFn() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("api_key", c.ApiKey)
		req.Header.Add("Hyperstack-Client", fmt.Sprintf("csi-hyperstack/%s", c.Version))
		req.Header.Add("User-Agent", fmt.Sprintf("csi-hyperstack/%s (Go/%s; %s/%s)",
			c.Version, strings.TrimPrefix(runtime.Version(), "go"), runtime.GOOS, runtime.GOARCH))
		return nil
	}
}
