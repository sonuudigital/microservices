package handlers

import (
	"api-gateway/internal/middlewares"
	"net/http"
	"net/http/httputil"
	"net/url"
	"shared/logs"
)

func NewProxyHandler(targetURL string, logger logs.Logger) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		claims, ok := middlewares.GetUserClaims(req)
		if !ok {
			logger.Warn("claims not found in context")
			return
		}

		req.Header.Set("X-User-Email", claims.Email)
		req.Header.Set("X-User-ID", claims.Subject)

		req.Header.Del("Authorization")
	}

	return proxy, nil
}
