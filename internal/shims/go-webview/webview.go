package webview

import (
	"errors"
	"time"
)

type Option func(*Webview)

func WithDebugURL(url string) Option {
	return func(w *Webview) { w.debugURL = url }
}

func WithTimeout(timeout time.Duration) Option {
	return func(w *Webview) { w.timeout = timeout }
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Element struct {
	NodeID      int
	TagName     string
	Attributes  map[string]string
	BoundingBox *BoundingBox
}

type ConsoleMessage struct {
	Type      string
	Text      string
	Timestamp time.Time
	URL       string
	Line      int
}

type Webview struct {
	debugURL string
	timeout  time.Duration
	console  []ConsoleMessage
}

func New(opts ...Option) (*Webview, error) {
	w := &Webview{timeout: 30 * time.Second}
	for _, opt := range opts {
		if opt != nil {
			opt(w)
		}
	}
	return w, nil
}

func (w *Webview) Close() error { return nil }

func (w *Webview) Navigate(string) error { return errors.New("webview backend unavailable") }

func (w *Webview) Click(string) error { return errors.New("webview backend unavailable") }

func (w *Webview) Type(string, string) error { return errors.New("webview backend unavailable") }

func (w *Webview) QuerySelector(string) (*Element, error) {
	return nil, errors.New("element not found")
}

func (w *Webview) QuerySelectorAll(string) ([]Element, error) {
	return nil, errors.New("webview backend unavailable")
}

func (w *Webview) GetConsole() []ConsoleMessage {
	out := make([]ConsoleMessage, len(w.console))
	copy(out, w.console)
	return out
}

func (w *Webview) ClearConsole() { w.console = nil }

func (w *Webview) Evaluate(string) (any, error) {
	return nil, errors.New("webview backend unavailable")
}

func (w *Webview) Screenshot() ([]byte, error) {
	return nil, errors.New("webview backend unavailable")
}
