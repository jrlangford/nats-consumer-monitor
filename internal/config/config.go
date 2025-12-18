package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nats-io/nats.go"
)

// ConsumerRef identifies a NATS JetStream consumer to monitor.
type ConsumerRef struct {
	Stream   string `json:"stream"`
	Consumer string `json:"consumer"`
}

// Config holds the application configuration.
type Config struct {
	Consumers []ConsumerRef
}

// Load reads the consumer configuration from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read consumers config %s: %w", path, err)
	}

	// Try parsing as object with "consumers" key first
	var cfg struct {
		Consumers []ConsumerRef `json:"consumers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Fallback: try parsing as plain array
		var list []ConsumerRef
		if errList := json.Unmarshal(data, &list); errList != nil {
			return nil, fmt.Errorf("parse consumers config %s: %w", path, err)
		}
		cfg.Consumers = list
	}

	if len(cfg.Consumers) == 0 {
		return nil, fmt.Errorf("no consumers configured in %s", path)
	}

	return &Config{Consumers: cfg.Consumers}, nil
}

// natsContext represents the NATS CLI context file format.
type natsContext struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	ServerURL   string   `json:"server_url"`
	Servers     []string `json:"servers"`
	Token       string   `json:"token"`
	User        string   `json:"user"`
	Pass        string   `json:"pass"`
	Password    string   `json:"password"`
	Creds       string   `json:"creds"`
	NKey        string   `json:"nkey"`
	Cert        string   `json:"cert"`
	Key         string   `json:"key"`
	CA          string   `json:"ca"`
}

// LoadNATSFromContext loads NATS connection settings from the NATS CLI context.
func LoadNATSFromContext() (string, []nats.Option, error) {
	ctxName := os.Getenv("NATS_CONTEXT")
	if ctxName == "" {
		return "", nil, fmt.Errorf("NATS_CONTEXT is not set")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil, fmt.Errorf("resolve home directory: %w", err)
	}

	contextPath := filepath.Join(home, ".config", "nats", "context", ctxName+".json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return "", nil, fmt.Errorf("read NATS context %s: %w", ctxName, err)
	}

	var ctx natsContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return "", nil, fmt.Errorf("parse NATS context %s: %w", ctxName, err)
	}

	natsURL := firstNonEmpty(ctx.URL, ctx.ServerURL)
	if natsURL == "" && len(ctx.Servers) > 0 {
		natsURL = ctx.Servers[0]
	}
	if natsURL == "" {
		return "", nil, fmt.Errorf("NATS context %s is missing a server URL", ctxName)
	}

	var opts []nats.Option
	if len(ctx.Servers) > 0 {
		opts = append(opts, withServers(ctx.Servers))
	}

	switch {
	case ctx.Creds != "":
		opts = append(opts, nats.UserCredentials(ctx.Creds))
	case ctx.Token != "":
		opts = append(opts, nats.Token(ctx.Token))
	case ctx.User != "":
		opts = append(opts, nats.UserInfo(ctx.User, firstNonEmpty(ctx.Pass, ctx.Password)))
	}

	if ctx.Cert != "" && ctx.Key != "" {
		opts = append(opts, nats.ClientCert(ctx.Cert, ctx.Key))
	}

	if ctx.CA != "" {
		opts = append(opts, nats.RootCAs(ctx.CA))
	}

	return natsURL, opts, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func withServers(servers []string) nats.Option {
	return func(o *nats.Options) error {
		o.Servers = append([]string{}, servers...)
		return nil
	}
}
