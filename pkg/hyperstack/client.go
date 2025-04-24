package hyperstack

import (
	"context"
	"net/http"
)

type HyperstackClient struct {
	Client    *http.Client
	ApiKey    string
	ApiServer string
}

func NewHyperstackClient(
	apiKey string,
	apiServer string,
) *HyperstackClient {
	return &HyperstackClient{
		Client:    http.DefaultClient,
		ApiKey:    apiKey,
		ApiServer: apiServer,
	}
}

func (c HyperstackClient) GetAddHeadersFn() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("api_key", c.ApiKey)
		return nil
	}
}
