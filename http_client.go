package heimdall

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gojektech/valkyrie"
	"github.com/pkg/errors"
)

const defaultRetryCount int = 0

type httpClient struct {
	client *http.Client

	retryCount int
	retrier    Retriable
}

// NewHTTPClient returns a new instance of HTTPClient
func NewHTTPClient(timeoutInMilliseconds int) Client {
	httpTimeout := time.Duration(timeoutInMilliseconds) * time.Millisecond
	return &httpClient{
		client: &http.Client{
			Timeout: httpTimeout,
		},

		retryCount: defaultRetryCount,
		retrier:    NewNoRetrier(),
	}
}

// SetRetryCount sets the retry count for the httpClient
func (c *httpClient) SetRetryCount(count int) {
	c.retryCount = count
}

// SetRetrier sets the strategy for retrying
func (c *httpClient) SetRetrier(retrier Retriable) {
	c.retrier = retrier
}

// Get makes a HTTP GET request to provided URL
func (c *httpClient) Get(url string, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return response, errors.Wrap(err, "GET - request creation failed")
	}

	request.Header = headers

	return c.Do(request)
}

// Post makes a HTTP POST request to provided URL and requestBody
func (c *httpClient) Post(url string, body io.Reader, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return response, errors.Wrap(err, "POST - request creation failed")
	}

	request.Header = headers

	return c.Do(request)
}

// Put makes a HTTP PUT request to provided URL and requestBody
func (c *httpClient) Put(url string, body io.Reader, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return response, errors.Wrap(err, "PUT - request creation failed")
	}

	request.Header = headers

	return c.Do(request)
}

// Patch makes a HTTP PATCH request to provided URL and requestBody
func (c *httpClient) Patch(url string, body io.Reader, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPatch, url, body)
	if err != nil {
		return response, errors.Wrap(err, "PATCH - request creation failed")
	}

	request.Header = headers

	return c.Do(request)
}

// Delete makes a HTTP DELETE request with provided URL
func (c *httpClient) Delete(url string, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return response, errors.Wrap(err, "DELETE - request creation failed")
	}

	request.Header = headers

	return c.Do(request)
}

// Do makes an HTTP request with the native `http.Do` interface
func (c *httpClient) Do(request *http.Request) (*http.Response, error) {
	request.Close = true

	multiErr := &valkyrie.MultiError{}
	var response *http.Response

	for i := 0; i <= c.retryCount; i++ {
		var err error
		response, err = c.client.Do(request)
		if err != nil {
			multiErr.Push(err.Error())

			backoffTime := c.retrier.NextInterval(i)
			time.Sleep(backoffTime)
			continue
		}

		if response.StatusCode >= http.StatusInternalServerError {
			multiErr.Push(fmt.Sprintf("server error: %d", response.StatusCode))

			backoffTime := c.retrier.NextInterval(i)
			time.Sleep(backoffTime)
			fmt.Println("R: ", response.StatusCode)
			continue
		}

		multiErr = &valkyrie.MultiError{} // Clear errors if any iteration succeeds
		break
	}

	return response, multiErr.HasError()
}
