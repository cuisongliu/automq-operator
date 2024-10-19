/*
Copyright 2023 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	httpl "net/http"
	"time"
)

func httpTransport() httpl.Transport {
	return httpl.Transport{
		Proxy: httpl.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   200,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 2 * time.Minute,
		DisableCompression:    false,
		DisableKeepAlives:     false,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
}

type RespReturn struct {
	Data   []byte
	Code   int32
	Error  error
	Header httpl.Header
}

func RestHttpApi(ctx context.Context, url, method string, body io.Reader, timeout int32, fns ...func(h httpl.Header)) RespReturn {
	if timeout == 0 {
		timeout = 60
	}
	trans := httpTransport()
	// https://github.com/golang/go/issues/13801
	client := &httpl.Client{
		Transport: &trans,
	}
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(int64(timeout)*int64(time.Second)))
	defer cancel()
	req, _ := httpl.NewRequestWithContext(ctx, method, url, body)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("User-agent", "RealHttpConnector")
	req.Header.Set("Connection", "keep-alive")
	for _, f := range fns {
		f(req.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		return RespReturn{
			Data:   nil,
			Code:   500,
			Error:  err,
			Header: nil,
		}
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return RespReturn{
			Data:   nil,
			Code:   500,
			Error:  err,
			Header: nil,
		}
	}
	return RespReturn{
		Data:   data,
		Code:   int32(resp.StatusCode),
		Error:  err,
		Header: resp.Header,
	}
}
