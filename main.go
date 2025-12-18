package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nats-io/nats.go"
	"github.com/rivo/tview"
)

type ConsumerRef struct {
	Stream   string
	Consumer string
}

type consumerConfig struct {
	Consumers []ConsumerRef `json:"consumers"`
}

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

func main() {

	configPath := os.Getenv("CONSUMERS_CONFIG")
	if configPath == "" {
		configPath = "consumers.json"
	}

	consumers, err := loadConsumers(configPath)
	if err != nil {
		log.Fatal(err)
	}

	natsURL, natsOpts, err := loadNATSFromContext()
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(natsURL, natsOpts...)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()
	darkBg := tcell.NewRGBColor(24, 24, 37)
	border := tcell.NewRGBColor(88, 91, 112)
	//title := tcell.NewRGBColor(137, 180, 250)
	//fg := tcell.ColorWhite

	// Set application-wide background
	grid := tview.NewGrid().
		SetRows(0, 0, 0, 0).
		SetColumns(0, 0, 0, 0)
	grid.SetBackgroundColor(darkBg)

	views := make([]*tview.TextView, len(consumers))

	for i, c := range consumers {
		tv := tview.NewTextView()
		tv.SetDynamicColors(true)
		tv.SetBackgroundColor(darkBg)
		tv.SetBorder(true)
		tv.SetBorderColor(border)
		tv.SetTitle(fmt.Sprintf(" %s ", c.Consumer))
		//tv.SetTitleColor(title)

		views[i] = tv

		row := i / 4
		col := i % 4

		grid.AddItem(tv, row, col, 1, 1, 0, 0, false)
	}

	// Update loop
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for i, c := range consumers {
				ci, err := js.ConsumerInfo(c.Stream, c.Consumer)
				if err != nil {
					app.QueueUpdateDraw(func() {
						views[i].SetText(fmt.Sprintf("[red]ERROR[-]\n%v", err))
					})
					continue
				}

				text := fmt.Sprintf(
					"[yellow]Delivered:[-] %d\n"+
						"[yellow]Ack Floor:[-] %d\n"+
						"[yellow]Pending:[-] %d\n"+
						"[yellow]Redelivered:[-] %d\n"+
						"[yellow]Ack Pending:[-] %d\n",
					ci.Delivered.Consumer,
					ci.AckFloor.Consumer,
					ci.NumPending,
					ci.NumRedelivered,
					ci.NumAckPending,
				)

				app.QueueUpdateDraw(func() {
					views[i].SetText(text)
				})
			}
		}
	}()

	if err := app.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
		log.Fatal(err)
	}
}

func loadConsumers(path string) ([]ConsumerRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read consumers config %s: %w", path, err)
	}

	var cfg consumerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		var list []ConsumerRef
		if errList := json.Unmarshal(data, &list); errList != nil {
			return nil, fmt.Errorf("parse consumers config %s: %w", path, err)
		}
		cfg.Consumers = list
	}

	if len(cfg.Consumers) == 0 {
		return nil, fmt.Errorf("no consumers configured in %s", path)
	}

	return cfg.Consumers, nil
}

func loadNATSFromContext() (string, []nats.Option, error) {
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
		//opts = append(opts, nats.Servers(ctx.Servers...))
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
