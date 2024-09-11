package http_requests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/langgenius/dify-plugin-daemon/internal/utils/parser"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/routine"
	"github.com/langgenius/dify-plugin-daemon/internal/utils/stream"
)

func parseJsonBody(resp *http.Response, ret interface{}) error {
	defer resp.Body.Close()
	json_decoder := json.NewDecoder(resp.Body)
	return json_decoder.Decode(ret)
}

func RequestAndParse[T any](client *http.Client, url string, method string, options ...HttpOptions) (*T, error) {
	var ret T

	// check if ret is a map, if so, create a new map
	if _, ok := any(ret).(map[string]any); ok {
		ret = *new(T)
	}

	resp, err := Request(client, url, method, options...)
	if err != nil {
		return nil, err
	}

	// get read timeout
	read_timeout := int64(60000)
	for _, option := range options {
		if option.Type == "read_timeout" {
			read_timeout = option.Value.(int64)
			break
		}
	}
	time.AfterFunc(time.Millisecond*time.Duration(read_timeout), func() {
		// close the response body if timeout
		resp.Body.Close()
	})

	err = parseJsonBody(resp, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func GetAndParse[T any](client *http.Client, url string, options ...HttpOptions) (*T, error) {
	return RequestAndParse[T](client, url, "GET", options...)
}

func PostAndParse[T any](client *http.Client, url string, options ...HttpOptions) (*T, error) {
	return RequestAndParse[T](client, url, "POST", options...)
}

func PutAndParse[T any](client *http.Client, url string, options ...HttpOptions) (*T, error) {
	return RequestAndParse[T](client, url, "PUT", options...)
}

func DeleteAndParse[T any](client *http.Client, url string, options ...HttpOptions) (*T, error) {
	return RequestAndParse[T](client, url, "DELETE", options...)
}

func PatchAndParse[T any](client *http.Client, url string, options ...HttpOptions) (*T, error) {
	return RequestAndParse[T](client, url, "PATCH", options...)
}

func RequestAndParseStream[T any](client *http.Client, url string, method string, options ...HttpOptions) (*stream.Stream[T], error) {
	resp, err := Request(client, url, method, options...)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		error_text, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code: %d and respond with: %s", resp.StatusCode, error_text)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		error_text, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status code: %d and respond with: %s", resp.StatusCode, error_text)
	}

	ch := stream.NewStream[T](1024)

	// get read timeout
	read_timeout := int64(60000)
	raise_error_when_stream_data_not_match := false
	for _, option := range options {
		if option.Type == "read_timeout" {
			read_timeout = option.Value.(int64)
			break
		} else if option.Type == "raiseErrorWhenStreamDataNotMatch" {
			raise_error_when_stream_data_not_match = option.Value.(bool)
		}
	}
	time.AfterFunc(time.Millisecond*time.Duration(read_timeout), func() {
		// close the response body if timeout
		resp.Body.Close()
	})

	routine.Submit(func() {
		scanner := bufio.NewScanner(resp.Body)
		defer resp.Body.Close()

		for scanner.Scan() {
			data := scanner.Bytes()
			if len(data) == 0 {
				continue
			}

			if bytes.HasPrefix(data, []byte("data: ")) {
				// split
				data = data[6:]
			}

			// unmarshal
			t, err := parser.UnmarshalJsonBytes[T](data)
			if err != nil {
				if raise_error_when_stream_data_not_match {
					ch.WriteError(err)
					break
				}
				continue
			}

			ch.Write(t)
		}

		ch.Close()
	})

	return ch, nil
}

func GetAndParseStream[T any](client *http.Client, url string, options ...HttpOptions) (*stream.Stream[T], error) {
	return RequestAndParseStream[T](client, url, "GET", options...)
}

func PostAndParseStream[T any](client *http.Client, url string, options ...HttpOptions) (*stream.Stream[T], error) {
	return RequestAndParseStream[T](client, url, "POST", options...)
}

func PutAndParseStream[T any](client *http.Client, url string, options ...HttpOptions) (*stream.Stream[T], error) {
	return RequestAndParseStream[T](client, url, "PUT", options...)
}
