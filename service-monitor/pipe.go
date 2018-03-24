package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

type LogPipe struct {
	Chan   chan []byte
	Cancel chan time.Time

	buf []byte
}

func (lp *LogPipe) Write(p []byte) (n int, err error) {
	bs := []byte(search)

	lp.buf = append(lp.buf, p...)

	for bytes.Contains(lp.buf, []byte("\n")) {
		i := bytes.Index(lp.buf, []byte("\n"))
		s := lp.buf[0 : i+1]
		lp.buf = lp.buf[i+1:]

		if bytes.Contains(s, bs) {
			lp.Chan <- s
		}
	}

	return len(p), nil
}

func logPipe(matches []sdjournal.Match, filter []sdjournal.Match) *LogPipe {
	for _, f := range filter {
		matches = append(matches, f)
	}

	lp := LogPipe{
		Chan:   make(chan []byte, 1024),
		Cancel: make(chan time.Time),
	}

	go func() {
		r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
			// NumFromTail: 2048,
			Since:     time.Duration(-24) * time.Hour,
			Matches:   matches,
			Formatter: logFormatter,
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

func pipeReader(r *LogPipe, w io.Writer) {
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
				time.Sleep(50 * time.Millisecond)
			}
		}

		if len(buf) > 0 {
			w.Write(buf)
		}
	}
}

func logFormatter(entry *sdjournal.JournalEntry) (string, error) {
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
}
