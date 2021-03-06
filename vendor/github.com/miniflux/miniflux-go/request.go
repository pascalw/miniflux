// Copyright 2017 Frédéric Guillot. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package miniflux

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	userAgent      = "Miniflux Client Library <https://github.com/miniflux/miniflux-go>"
	defaultTimeout = 80
)

// List of exposed errors.
var (
	ErrNotAuthorized = errors.New("miniflux: unauthorized (bad credentials)")
	ErrForbidden     = errors.New("miniflux: access forbidden")
	ErrServerError   = errors.New("miniflux: internal server error")
	ErrNotFound      = errors.New("miniflux: resource not found")
)

type errorResponse struct {
	ErrorMessage string `json:"error_message"`
}

type request struct {
	endpoint string
	username string
	password string
}

func (r *request) Get(path string) (io.ReadCloser, error) {
	return r.execute(http.MethodGet, path, nil)
}

func (r *request) Post(path string, data interface{}) (io.ReadCloser, error) {
	return r.execute(http.MethodPost, path, data)
}

func (r *request) PostFile(path string, f io.ReadCloser) (io.ReadCloser, error) {
	return r.execute(http.MethodPost, path, f)
}

func (r *request) Put(path string, data interface{}) (io.ReadCloser, error) {
	return r.execute(http.MethodPut, path, data)
}

func (r *request) Delete(path string) (io.ReadCloser, error) {
	return r.execute(http.MethodDelete, path, nil)
}

func (r *request) execute(method, path string, data interface{}) (io.ReadCloser, error) {
	if r.endpoint[len(r.endpoint)-1:] == "/" {
		r.endpoint = r.endpoint[:len(r.endpoint)-1]
	}

	u, err := url.Parse(r.endpoint + path)
	if err != nil {
		return nil, err
	}

	request := &http.Request{
		URL:    u,
		Method: method,
		Header: r.buildHeaders(),
	}
	request.SetBasicAuth(r.username, r.password)

	if data != nil {
		switch data.(type) {
		case io.ReadCloser:
			request.Body = data.(io.ReadCloser)
		default:
			request.Body = ioutil.NopCloser(bytes.NewBuffer(r.toJSON(data)))
		}
	}

	client := r.buildClient()
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrNotAuthorized
	case http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusInternalServerError:
		return nil, ErrServerError
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusBadRequest:
		defer response.Body.Close()

		var resp errorResponse
		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(&resp); err != nil {
			return nil, fmt.Errorf("miniflux: bad request error (%v)", err)
		}

		return nil, fmt.Errorf("miniflux: bad request (%s)", resp.ErrorMessage)
	}

	if response.StatusCode > 400 {
		return nil, fmt.Errorf("miniflux: status code=%d", response.StatusCode)
	}

	return response.Body, nil
}

func (r *request) buildClient() http.Client {
	return http.Client{
		Timeout: time.Duration(defaultTimeout * time.Second),
	}
}

func (r *request) buildHeaders() http.Header {
	headers := make(http.Header)
	headers.Add("User-Agent", userAgent)
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	return headers
}

func (r *request) toJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("Unable to convert interface to JSON:", err)
		return []byte("")
	}

	return b
}
