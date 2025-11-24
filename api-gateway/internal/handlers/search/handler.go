package search

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	ErrTargetURLNil = errors.New("target URL is nil")
)

func NewSearchHandler(targetURL *url.URL) (http.Handler, error) {
	if targetURL == nil {
		return nil, ErrTargetURLNil
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	return proxy, nil
}
