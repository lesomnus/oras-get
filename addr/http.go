package addr

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/lesomnus/z"
)

type Http string

func (a *Http) Evaluate() error {
	if a == nil || *a == "" {
		return errors.New("address is empty")
	}

	scheme, host, port := a.Split()
	z.FallbackP(&scheme, "http")
	switch scheme {
	case "http", "https":
		z.FallbackP(&host, "0.0.0.0")
	case "unix":
		z.FallbackP(&host, "oras-get.sock")
		port = ""
	default:
		return fmt.Errorf("unsupported scheme: %q", scheme)
	}

	if port != "" {
		port = ":" + port
	}
	*a = Http(fmt.Sprintf("%s://%s%s", scheme, host, port))

	return nil
}

func (a Http) Split() (scheme string, host string, port string) {
	v := string(a)
	if i := strings.Index(v, "://"); i >= 0 {
		scheme = v[:i]
		v = v[i+3:]
	}

	es := strings.SplitN(v, ":", 2)
	host = es[0]
	if len(es) > 1 {
		port = es[1]
	}

	return
}

func (a Http) Scheme() string {
	v, _, _ := a.Split()
	return v
}

func (a Http) Host() string {
	_, v, _ := a.Split()
	return v
}

func (a Http) Port() string {
	_, _, v := a.Split()
	return v
}

func (a Http) Normalized() Http {
	scheme, host, port := a.Split()
	return a.normalize(scheme, host, port)
}

func (Http) normalize(scheme, host, port string) Http {
	v := scheme
	if v != "" {
		v += "://"
	}
	v += host
	if port != "" {
		v += ":" + port
	}
	return Http(v)
}

func (a Http) Target() string {
	scheme, host, port := a.Split()
	if scheme == "" {
		scheme = "http"
	}
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	if port != "" {
		port = ":" + port
	}
	return scheme + "://" + host + port
}

func (a Http) WithHost(host string) Http {
	scheme, _, port := a.Split()
	return a.normalize(scheme, host, port)
}

func (a Http) WithPort(port string) Http {
	scheme, host, _ := a.Split()
	return a.normalize(scheme, host, port)
}

func (a Http) HostPort() string {
	_, host, port := a.Split()
	return net.JoinHostPort(host, port)
}

func (a Http) Listen() (net.Listener, error) {
	scheme, host, port := a.Split()
	switch scheme {
	case "http", "https":
		return net.Listen("tcp", net.JoinHostPort(host, port))
	case "unix":
		return net.Listen("unix", host)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", scheme)
	}
}
