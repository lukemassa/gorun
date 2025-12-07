package client

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

type commandRequest struct {
	Cmd string
	Env []string
}

func NewClient(sock string) *Client {

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

func (c *Client) GetCommand(cmd string, env []string) (string, error) {

	requestContent := commandRequest{
		Cmd: cmd,
		Env: env,
	}
	b, err := json.Marshal(requestContent)
	if err != nil {
		return "", err
	}

	// URL host is ignored — must be syntactically valid, but irrelevant.
	req, err := http.NewRequest("POST", "http://unix/command", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
