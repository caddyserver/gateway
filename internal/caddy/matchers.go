// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package caddy

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/caddyserver/gateway/internal/caddyv2/caddyhttp"
)

// getPathMatcher .
// ref; https://caddyserver.com/docs/json/apps/http/servers/routes/match/path/
func (i *Input) getPathMatcher(matcher *caddyhttp.Match, path *gatewayv1.HTTPPathMatch) error {
	if path == nil || path.Value == nil {
		return nil
	}
	value := *path.Value
	if value == "" {
		return nil
	}
	var matchType gatewayv1.PathMatchType
	if path.Type == nil {
		matchType = gatewayv1.PathMatchPathPrefix
	} else {
		matchType = *path.Type
	}

	// If the path is `/` and the match type is a PathPrefix,
	// ignore it. This is just a verbose way of saying "match
	// all paths".
	if value == "/" && matchType == gatewayv1.PathMatchPathPrefix {
		return nil
	}

	switch matchType {
	case gatewayv1.PathMatchExact:
		matcher.Path = caddyhttp.MatchPath{value}
	case gatewayv1.PathMatchPathPrefix:
		matcher.Path = caddyhttp.MatchPath{value + "*"}
	case gatewayv1.PathMatchRegularExpression:
		matcher.PathRE = &caddyhttp.MatchPathRE{
			MatchRegexp: caddyhttp.MatchRegexp{
				Pattern: value,
			},
		}
	}
	return nil
}

// getHeaderMatcher .
// ref; https://caddyserver.com/docs/json/apps/http/servers/routes/match/header/
func (i *Input) getHeaderMatcher(matcher *caddyhttp.Match, v []gatewayv1.HTTPHeaderMatch) error {
	if v == nil {
		return nil
	}

	// TODO: implement
	return nil
}

// getQueryMatcher .
// ref; https://caddyserver.com/docs/json/apps/http/servers/routes/match/query/
func (i *Input) getQueryMatcher(matcher *caddyhttp.Match, v []gatewayv1.HTTPQueryParamMatch) error {
	if v == nil {
		return nil
	}

	// TODO: implement
	return nil
}

// getMethodMatcher .
// ref; https://caddyserver.com/docs/json/apps/http/servers/routes/match/method/
func (i *Input) getMethodMatcher(matcher *caddyhttp.Match, m *gatewayv1.HTTPMethod) error {
	if m == nil {
		return nil
	}
	matcher.Method = caddyhttp.MatchMethod{string(*m)}
	return nil
}
