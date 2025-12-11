package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	log "github.com/lukemassa/clilog"
	"github.com/lukemassa/gorun/internal/server"
)

type Client struct {
	httpClient *http.Client
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

	requestContent := server.ExecutableRequest{
		MainPackage: cmd,
		Env:         env,
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
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("got %d calling API: %s", resp.StatusCode, string(body))
	}
	var commandResponse server.ExecutableResponse
	err = json.Unmarshal(body, &commandResponse)
	if err != nil {
		return "", err
	}
	if commandResponse.Executable == "" {
		return "", fmt.Errorf("failed to compile: %s", commandResponse)
	}

	return commandResponse.Executable, nil
}

func (c *Client) DeleteCommand(cmd string, env []string) error {

	// TODO: Dedupe from above
	requestContent := server.ExecutableRequest{
		MainPackage: cmd,
		Env:         env,
	}
	b, err := json.Marshal(requestContent)
	if err != nil {
		return err
	}

	// URL host is ignored — must be syntactically valid, but irrelevant.
	req, err := http.NewRequest("DELETE", "http://unix/command", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("got %d calling API: %s", resp.StatusCode, string(body))
	}
	log.Debugf("Deleted response: %s", string(body))
	return nil
}
