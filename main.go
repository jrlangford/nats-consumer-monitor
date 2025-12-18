package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nats-io/nats.go"
	"github.com/rivo/tview"
)

type ConsumerRef struct {
	Stream   string
	Consumer string
}

func main() {

	consumers := []ConsumerRef{
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
	}

	// NATS connection
	natsURL := "nats://localhost:30222"
	nc, err := nats.Connect(
		natsURL,
		nats.UserInfo("js", "js"),
	)
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
