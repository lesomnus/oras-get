package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/og"
	"github.com/lesomnus/z"
	"gopkg.in/yaml.v3"
)

var use = z.NewUse[*Config]()

func Into(ctx context.Context, v *Config) context.Context {
	return use.Into(ctx, v)
}

func From(ctx context.Context) *Config {
	return use.Must(ctx)
}

type Config struct {
	Server     ServerConfig
	Registries map[string]RegistryConfig
	Routes     []RouteConfig

	Otel OtelConfig

	filepath string
}

func FromPath(p string) (*Config, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	var c Config
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	c.filepath = p
	return &c, nil
}

func (c *Config) Evaluate() error {
	err_registries := []error{}
	for k, v := range c.Registries {
		if err := v.Evaluate(); err != nil {
			err_registries = append(err_registries, fmt.Errorf(".[%q]%w", k, err))
		}
	}

	err_routes := []error{}
	for i := range c.Routes {
		if err := c.Routes[i].Evaluate(); err != nil {
			err_routes = append(err_routes, fmt.Errorf(".[%d]%w", i, err))
		}
	}

	return errors.Join(
		z.CatErr(".server", c.Server.Evaluate()),
		z.CatErr(".registries", errors.Join(err_registries...)),
		z.CatErr(".routes", errors.Join(err_routes...)),
		z.CatErr(".otel", c.Otel.Evaluate()),
	)
}

func (c *Config) NewRouter() (*og.Router, error) {
	r := &og.Router{
		Upstreams: map[string]og.Server{},
		Matchers:  []match.Matcher{},
	}
	for k, v := range c.Registries {
		registry, err := v.Build(v.Addr.HostPort())
		if err != nil {
			return nil, z.Err(err, "build registry %q", k)
		}
		r.Upstreams[k] = og.NewServer(registry)
	}
	for i, v := range c.Routes {
		m, err := v.Build()
		if err != nil {
			return nil, z.Err(err, "build route matcher [%d]", i)
		}
		r.Matchers = append(r.Matchers, m)
	}

	return r, nil
}
