package config

import (
	"errors"
	"strings"
	"text/template"

	"github.com/lesomnus/oras-get/match"
)

type RouteConfig struct {
	Domain *string
	Repo   *string

	Next string
}

func (c *RouteConfig) Evaluate() error {
	return nil
}

func (c *RouteConfig) Build() (match.Matcher, error) {
	if c.Next == "" {
		return nil, errors.New("next field is required")
	}
	if c.Domain != nil || c.Repo != nil {
		return &match.StaticNamedMatcher{
			Domain: c.Domain,
			Repo:   c.Repo,
			Next:   c.Next,
		}, nil
	}

	if !strings.Contains(c.Next, "{{") {
		return match.FixedMatcher(c.Next), nil
	}

	tmpl, err := template.New("").Parse(c.Next)
	if err != nil {
		return nil, err
	}
	return &match.TemplateMatcher{Next: tmpl}, nil
}
