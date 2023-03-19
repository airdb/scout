package lokikit

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	lokiApiPush = "/loki/api/v1/push"
)

// LokiWriter send log messages towards Loki.
type LokiWriter struct {
	io.Writer
	config    *lokiClientConfig
	quit      chan struct{}
	entries   chan lokiEntry
	waitGroup sync.WaitGroup
	client    *lokiClient
}

type lokiPushRequest struct {
	Streams []lokiStreamAdapter `json:"streams"`
}

type lokiStreamAdapter struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiEntry struct {
	Timestamp time.Time
	Line      []byte
	Content   string
}

type LokiWriterOption func(h *LokiWriter)

func WithBasicAuth(username, password string) LokiWriterOption {
	return func(h *LokiWriter) {
		h.client.beforeDos = append(h.client.beforeDos, func(r *http.Request) {
			if len(username) > 0 && len(password) > 0 {
				r.SetBasicAuth(username, password)
			}
		})
	}
}

func WithLabels(kv map[string]string) LokiWriterOption {
	return func(h *LokiWriter) {
		for k, v := range kv {
			h.config.Labels[k] = v
		}
	}
}

func NewLokiWriter(
	rootUrl string, timeOffset int64, opts ...LokiWriterOption,
) (*LokiWriter, error) {
	conf := &lokiClientConfig{
		PushURL:            strings.TrimSuffix(rootUrl, "/") + lokiApiPush,
		BatchWait:          time.Second * 2,
		BatchEntriesNumber: 1024,
		Labels:             map[string]string{},
	}
	writer := &LokiWriter{
		config:  conf,
		quit:    make(chan struct{}),
		entries: make(chan lokiEntry, LOG_ENTRIES_CHAN_SIZE),
		client:  newLokiClient(),
	}

	for _, opt := range opts {
		opt(writer)
	}

	writer.waitGroup.Add(1)
	go writer.run()

	return writer, nil
}

var _ io.Writer = &LokiWriter{}

func (h *LokiWriter) Write(p []byte) (n int, err error) {
	h.entries <- lokiEntry{
		Line: append([]byte{}, p...),
	}

	return len(p), nil
}

func (h *LokiWriter) Shutdown() {
	close(h.quit)
	h.waitGroup.Wait()
}

func (h *LokiWriter) run() {
	var batch [][]string
	batchSize := 0
	defer func() {
		if batchSize > 0 {
			h.send(batch)
		}
		h.waitGroup.Done()
	}()

	maxWait := time.NewTimer(h.config.BatchWait)
	for {
		select {
		case <-h.quit:
			return
		case entry := <-h.entries:
			e := parseEntry(entry)
			batch = append(batch, []string{
				strconv.FormatInt(e.Timestamp.UnixNano(), 10),
				string(e.Content),
			})
			batchSize++
			if batchSize >= h.config.BatchEntriesNumber {
				h.send(batch)
				batch = nil
				batchSize = 0
				maxWait.Reset(h.config.BatchWait)
			}
		case <-maxWait.C:
			if batchSize > 0 {
				h.send(batch)
				batch = nil
				batchSize = 0
			}
			maxWait.Reset(h.config.BatchWait)
		}
	}
}

func (h *LokiWriter) send(entries [][]string) {
	req := lokiPushRequest{
		Streams: []lokiStreamAdapter{
			{
				Stream: h.config.Labels,
				Values: entries,
			},
		},
	}

	buf, err := json.Marshal(req)
	if err != nil {
		log.Printf("promtail.ClientProto: unable to marshal: %s\n", err)
		return
	}

	resp, body, err := h.client.sendJsonReq("POST", h.config.PushURL, "application/json", buf)
	if err != nil {
		log.Printf("promtail.ClientProto: unable to send an HTTP request: %s\n", err)
		return
	}

	if resp.StatusCode != 204 {
		log.Printf("promtail.ClientProto: Unexpected HTTP status code: %d, message: %s\n", resp.StatusCode, body)
		return
	}
}

func parseEntry(entry lokiEntry) lokiEntry {
	var evt map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(entry.Line))
	d.UseNumber()
	if err := d.Decode(&evt); err != nil {
		log.Printf("LokiWriter: Failed to parse log entry '%s': %s\n", entry.Line, err)
		return entry
	}

	var buf bytes.Buffer
	keys := make([]string, 0, len(evt))
	for k := range evt {
		switch k {
		case "time":
			if s, ok := evt["time"].(string); ok {
				ts, err := time.Parse(s, time.RFC3339Nano)
				if err == nil {
					entry.Timestamp = ts
				}
			}
		default:
			keys = append(keys, k)
		}
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	sort.Strings(keys)
	for _, k := range keys {
		buf.WriteByte(' ')
		buf.WriteString(k)
		buf.WriteByte('=')
		v := evt[k]
		switch tv := v.(type) {
		case string:
			if needsQuote(tv) {
				buf.WriteString(strconv.Quote(tv))
			} else {
				buf.WriteString(tv)
			}
		default:
			b, _ := json.Marshal(v)
			buf.Write(b)
		}
	}
	entry.Content = buf.String()

	return entry
}

// needsQuote returns true when the string s should be quoted in output.
func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}
