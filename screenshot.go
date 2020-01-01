package screenshot

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/emulation"
	chrome "go.ajitem.com/gcf/v2"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//TODO: Useragent, clipping, HTTP Basic Auth, Callback
type Opts struct {
	Url       string
	Width     int
	Height    int
	Delay     time.Duration
	UserAgent string
}

func NewOpts(r *http.Request) *Opts {
	o := &Opts{}

	o.Url = r.URL.Query().Get("url")

	if val, err := strconv.Atoi(r.URL.Query().Get("height")); err == nil {
		o.Height = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("width")); err == nil {
		o.Width = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("delay")); err == nil {
		o.Delay = time.Duration(val) * time.Millisecond
	}

	o.UserAgent = r.URL.Query().Get("useragent")

	return o
}

type Screenshot struct {
	Path    string
	Port    int
	Browser chrome.Browser
	Width   int
	Height  int
}

func NewScreenshot(path string, port int) *Screenshot {
	s := &Screenshot{
		Path: path, Port: port, Browser: &chrome.Chrome{},
		Width: 1920, Height: 1080,
	}

	err := s.launch()
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func (s *Screenshot) launch() (err error) {
	_, err = s.Browser.Launch(
		s.Path,
		chrome.Int(s.Port),
		chrome.StringSlice([]string{}),
	)

	go s.Browser.Wait()

	return
}

func (s *Screenshot) getScreenshots(opts *Opts) ([]byte, error) {
	if opts.Height == 0 {
		opts.Height = s.Height
	}

	if opts.Width == 0 {
		opts.Width = s.Width
	}

	tab, err := s.Browser.OpenNewTab(120 * time.Second)
	if err != nil {
		return nil, err
	}

	if opts.UserAgent != "" {
		tab.AttachHook(func(c *cdp.Client) error {
			return c.Emulation.SetUserAgentOverride(context.Background(), emulation.NewSetUserAgentOverrideArgs(opts.UserAgent))
		})
	}

	_, err = tab.Navigate(opts.Url, 120*time.Second)
	if err != nil {
		return nil, err
	}

	time.Sleep(opts.Delay)

	screenshot, err := tab.CaptureScreenshot(chrome.ScreenshotOpts{Width: opts.Width, Height: opts.Height}, 120*time.Second)
	if err != nil {
		return nil, err
	}

	rawImage, err := base64.StdEncoding.DecodeString(screenshot[strings.Index(screenshot, ",")+1:])
	if err != nil {
		return nil, fmt.Errorf("error: cannot decode base64 image: %v", err)
	}

	image, err := png.Decode(bytes.NewReader(rawImage))
	if err != nil {
		return nil, fmt.Errorf("error: bad png: %v", err)
	}

	var buf bytes.Buffer

	err = png.Encode(&buf, image)
	if err != nil {
		return nil, err
	}

	err = s.Browser.CloseTab(tab, 120*time.Second)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Screenshot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	opts := NewOpts(r)

	screenshot, err := s.getScreenshots(opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(200)
	_, _ = w.Write(screenshot)
}
