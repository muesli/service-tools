package main

import (
	"fmt"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type LogModel struct {
	Name        string
	Description string
	Matches     []sdjournal.Match
}

type LogPipe struct {
	Chan   chan []byte
	Cancel chan time.Time
}

func (lmw *LogPipe) Write(p []byte) (n int, err error) {
	lmw.Chan <- p
	return len(p), nil
}

var (
	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "monitor service logs",
		Long:  `The logs command starts an interactive service monitor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			form, err := logsForm()
			if err != nil {
				return err
			}

			if err = app.SetRoot(form, true).Run(); err != nil {
				return err
			}
			return apperr
		},
	}
)

func logsModel() ([]LogModel, error) {
	ts, err := services()
	if err != nil {
		return []LogModel{}, fmt.Errorf("Can't find systemd services: %s", err)
	}

	model := []LogModel{
		{
			Name:        "All",
			Description: "Everything in the log",
			Matches:     []sdjournal.Match{},
		},
		{
			Name:        "All Errors",
			Description: "All errors in the log",
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
					Value: "3",
				},
			},
		},
		{
			Name:        "All Warnigns",
			Description: "All warnings in the log",
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
					Value: "3",
				},
				{
					Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
					Value: "4",
				},
			},
		},
		{
			Name:        "Kernel",
			Description: "Kernel log",
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER,
					Value: "kernel",
				},
			},
		},
	}
	for _, service := range ts {
		model = append(model, LogModel{
			Name:        service.Name,
			Description: service.Description,
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
					Value: service.Name,
				},
			},
		})
	}

	return model, nil
}

func logsForm() (tview.Primitive, error) {
	model, err := logsModel()
	if err != nil {
		return nil, err
	}

	list := tview.NewList()
	logView := tview.NewTextView()
	errLogView := tview.NewTextView()

	list.
		SetBorder(true).
		SetTitle("Logs")
	list.
		SetDoneFunc(func() {
			app.SetFocus(logView)
		})

	logView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			var next tview.Primitive = errLogView
			switch key {
			case tcell.KeyBacktab:
				fallthrough
			case tcell.KeyEscape:
				next = list
			}
			app.SetFocus(next)
		})
	logView.
		SetBorder(true).
		SetTitle("Service Log")

	errLogView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			var next tview.Primitive = list
			switch key {
			case tcell.KeyBacktab:
				next = logView
			}
			app.SetFocus(next)
		})
	errLogView.
		SetBorder(true).
		SetTitle("Global Error Log")

	errReader := logPipe([]sdjournal.Match{
		{
			Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
			Value: "3",
		},
	})
	go pipeReader(errReader, errLogView)

	var mainReader *LogPipe
	list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		u := model[index]
		logView.Clear()
		logView.SetTitle(u.Name)

		if mainReader != nil {
			mainReader.Cancel <- time.Now()
		}
		mainReader = logPipe(u.Matches)
		go pipeReader(mainReader, logView)

		app.SetFocus(logView)
	})

	for _, srv := range model {
		list.AddItem(srv.Name, srv.Description, 0, nil)
	}

	flex := tview.NewFlex().
		AddItem(list, 32, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(logView, 0, 8, false).
			AddItem(errLogView, 8, 1, false), 0, 1, false)

	return flex, nil
}

func pipeReader(r *LogPipe, textView *tview.TextView) {
	for {
		var buf []byte
		done := false

		for !done {
			select {
			case l, ok := <-r.Chan:
				if !ok {
					return
				}
				buf = append(buf, l...)
			default:
				done = true
				break
			}
			time.Sleep(500 * time.Microsecond)
		}

		if len(buf) > 0 {
			textView.Write(buf)
			textView.ScrollToEnd()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func logPipe(matches []sdjournal.Match) *LogPipe {
	lp := LogPipe{
		Chan:   make(chan []byte),
		Cancel: make(chan time.Time),
	}

	go func() {
		r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
			Since:   time.Duration(-12) * time.Hour,
			Matches: matches,
			Formatter: func(entry *sdjournal.JournalEntry) (string, error) {
				color := "gray"
				switch entry.Fields["PRIORITY"] {
				case "0":
					fallthrough
				case "1":
					fallthrough
				case "2":
					fallthrough
				case "3":
					color = "red"
				case "4":
					color = "darkred"
				case "5":
					color = "silver"
				}
				return fmt.Sprintf("[green]%s [blue]%s [%s]%s\n",
					time.Unix(0, int64(entry.RealtimeTimestamp)*int64(time.Microsecond)).Format("Jan 02 15:04:05"),
					entry.Fields["SYSLOG_IDENTIFIER"],
					color,
					entry.Fields["MESSAGE"]), nil
			},
		})

		if err != nil {
			panic(err)
		}
		if r == nil {
			panic("journal reader is nil")
		}
		defer r.Close()
		defer close(lp.Chan)

		// and follow the reader synchronously
		if err = r.Follow(lp.Cancel, &lp); err != sdjournal.ErrExpired {
			panic(err)
		}
	}()

	return &lp
}

func init() {
	RootCmd.AddCommand(logsCmd)
}
