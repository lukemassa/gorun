package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// ignore "network" and "addr" from HTTP; always dial the socket
			return net.Dial("unix", sock)
		},
	}
	return &Client{
		httpClient: &http.Client{
			Transport: tr,
		},
	}
}

func (c *Client) GetBinary(cmd string, env []string) (string, error) {

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Get("http://unix/binary")
	// URL host is ignored — must be syntactically valid, but irrelevant.
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
