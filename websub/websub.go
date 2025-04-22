package websub

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// https://www.w3.org/TR/websub/

type WebSub struct {
	subscribeLink string

	serveMux   *http.ServeMux
	httpServer *http.Server
	httpClient *http.Client
}

type WebSubOpts struct {
	Pattern       string
	ClientTimeout time.Duration
}

func NewWebSub(subscribeLink, addr string, handleCallback func(w http.ResponseWriter, r *http.Request), opts *WebSubOpts) *WebSub {
	ws := &WebSub{
		subscribeLink: subscribeLink,
	}

	httpClient := &http.Client{}

	pattern := "/"
	if opts != nil {
		if opts.Pattern != "" {
			pattern = opts.Pattern
		}
		if opts.ClientTimeout != 0 {
			httpClient.Timeout = opts.ClientTimeout
		}
	}

	httpServer := &http.Server{
		Addr: addr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(pattern, handleCallback)
	httpServer.Handler = mux

	ws.httpServer = httpServer
	ws.httpClient = httpClient
	ws.serveMux = mux

	return ws
}

func (ws *WebSub) Run() error {
	return ws.httpServer.ListenAndServe()
}

type SubMode int8

const (
	ModeSubscribe SubMode = iota
	ModeUnsubscribe
	ModeDenied
)

func ParseSubMode(mode string) SubMode {
	switch mode {
	case "subscribe":
		return ModeSubscribe
	case "unsubscribe":
		return ModeUnsubscribe
	case "denied":
		return ModeDenied
	}
	return -1
}

func (mode SubMode) String() string {
	switch mode {
	case ModeSubscribe:
		return "subscribe"
	case ModeUnsubscribe:
		return "unsubscribe"
	case ModeDenied:
		return "denied"
	}
	return fmt.Sprintf("SubMode(%d)", mode)
}

type SubscribeOpts struct {
	LeaseSeconds uint32
	// Secret       string
}

func (ws *WebSub) Subscribe(mode SubMode, callbackUrl string, topicUrl string, opts *SubscribeOpts) error {
	if callbackUrl == "" {
		return fmt.Errorf("callbackUrl is empty")
	}
	if topicUrl == "" {
		return fmt.Errorf("topicUrl is empty")
	}

	formData := url.Values{
		"hub.callback":      {callbackUrl},
		"hub.mode":          {mode.String()},
		"hub.topic":         {topicUrl},
		"hub.lease_seconds": {"864000"},
	}

	if opts != nil {
		if opts.LeaseSeconds > 0 {
			formData.Set("hub.lease_seconds", strconv.FormatUint(uint64(opts.LeaseSeconds), 10))
		}
	}

	body := strings.NewReader(formData.Encode())

	req, err := http.NewRequest(http.MethodPost, ws.subscribeLink, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ws.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return errors.New(string(body))
	}

	return nil
}
