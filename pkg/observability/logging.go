package observability

import (
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"
)

type LogConfig struct {
	Host string
}

type LogstashWriter struct {
	host    string
	conn    net.Conn
	mu      sync.Mutex
	onError func(error)
}

func NewLogWriter(cfg LogConfig, onError func(error)) (io.Writer, error) {
	return &LogstashWriter{
		host:    cfg.Host,
		onError: onError,
	}, nil
}

func (w *LogstashWriter) connect() error {
	if w.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", w.host, time.Second*3)
	if err != nil {
		return err
	}
	w.conn = conn
	return nil
}

func (w *LogstashWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n = len(p)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		if w.onError != nil {
			w.onError(err)
		}
		return n, nil
	}

	if err := w.connect(); err != nil {
		if w.onError != nil {
			w.onError(err)
		}
		return n, nil
	}

	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		if w.onError != nil {
			w.onError(err)
		}
		return n, nil
	}
	logJSON = append(logJSON, '\n')

	deadline := time.Now().Add(time.Second * 3)
	if err := w.conn.SetWriteDeadline(deadline); err != nil {
		w.conn.Close()
		w.conn = nil
		if w.onError != nil {
			w.onError(err)
		}
		return n, nil
	}

	written := 0
	for written < len(logJSON) {
		var nw int
		nw, err = w.conn.Write(logJSON[written:])
		if err != nil {
			w.conn.Close()
			w.conn = nil
			if w.onError != nil {
				w.onError(err)
			}
			return n, nil
		}
		written += nw
	}

	return n, nil
}

func (w *LogstashWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}
