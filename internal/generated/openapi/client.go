// Package generated provides primitives to interact with the openapi HTTP API.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RequestEditorFn is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}


// SetCounterValue calls the POST /{actorId}/method/set endpoint
func (c *Client) SetCounterValue(ctx context.Context, actorId string, body SetValueRequest, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewSetCounterValueRequest(c.Server, actorId, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewSetCounterValueRequest generates requests for SetCounterValue
func NewSetCounterValueRequest(server string, actorId string, body SetValueRequest) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := strings.Replace("/{actorId}/method/set", "{actorId}", actorId, 1)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}


	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)


	req, err := http.NewRequest("POST", queryURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}


	req.Header.Add("Content-Type", "application/json")


	return req, nil
}

// DecrementCounter calls the POST /{actorId}/method/decrement endpoint
func (c *Client) DecrementCounter(ctx context.Context, actorId string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDecrementCounterRequest(c.Server, actorId)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewDecrementCounterRequest generates requests for DecrementCounter
func NewDecrementCounterRequest(server string, actorId string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := strings.Replace("/{actorId}/method/decrement", "{actorId}", actorId, 1)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}



	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}



	return req, nil
}

// GetCounterValue calls the GET /{actorId}/method/get endpoint
func (c *Client) GetCounterValue(ctx context.Context, actorId string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetCounterValueRequest(c.Server, actorId)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetCounterValueRequest generates requests for GetCounterValue
func NewGetCounterValueRequest(server string, actorId string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := strings.Replace("/{actorId}/method/get", "{actorId}", actorId, 1)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}



	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}



	return req, nil
}

// IncrementCounter calls the POST /{actorId}/method/increment endpoint
func (c *Client) IncrementCounter(ctx context.Context, actorId string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewIncrementCounterRequest(c.Server, actorId)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewIncrementCounterRequest generates requests for IncrementCounter
func NewIncrementCounterRequest(server string, actorId string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := strings.Replace("/{actorId}/method/increment", "{actorId}", actorId, 1)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}



	req, err := http.NewRequest("POST", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}



	return req, nil
}


func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}