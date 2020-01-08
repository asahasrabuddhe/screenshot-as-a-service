package screenshot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/emulation"
	"github.com/mafredri/cdp/protocol/network"
	chrome "go.ajitem.com/gcf/v2"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//TODO: Callback
type Opts struct {
	Url       string
	Top       int
	Left      int
	Width     int
	Height    int
	Delay     time.Duration
	UserAgent string
	FullPage  bool
	Username  string
	Password  string
}

func NewOpts(r *http.Request) *Opts {
	o := &Opts{}

	o.Url = r.URL.Query().Get("url")

	if val, err := strconv.Atoi(r.URL.Query().Get("top")); err == nil {
		o.Top = val
	}

	if val, err := strconv.Atoi(r.URL.Query().Get("left")); err == nil {
		o.Left = val
	}

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

	if r.URL.Query().Get("fullpage") == "true" {
		o.FullPage = true
	}

	o.Username = r.URL.Query().Get("username")
	o.Password = r.URL.Query().Get("password")

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

func (s *Screenshot) Terminate() error {
	return s.Browser.Terminate()
}

func (s *Screenshot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "/" {
		opts := NewOpts(r)

		screenshot, err := s.getScreenshot(opts)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(200)
		_, _ = w.Write(screenshot)
	} else {
		w.WriteHeader(404)
	}
}

func (s *Screenshot) getScreenshot(opts *Opts) ([]byte, error) {
	if opts.Height == 0 {
		opts.Height = s.Height
	}

	if opts.FullPage && (opts.Top == 0 || opts.Left == 0){
		opts.Height = 0
	} else {
		opts.FullPage = false
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

	if opts.Username != "" && opts.Password != "" {
		tab.AttachHook(func(c *cdp.Client) error {
			authHeader, err := json.Marshal(map[string]string{
				"Authorization": fmt.Sprintf(
					"Basic %s",
					base64.StdEncoding.EncodeToString(
						[]byte(fmt.Sprintf("%s:%s", opts.Username, opts.Password)),
					)),
			})
			if err != nil {
				return err
			}

			err = c.Network.Enable(context.Background(), network.NewEnableArgs())
			if err != nil {
				return err
			}

			return c.Network.SetExtraHTTPHeaders(context.Background(), network.NewSetExtraHTTPHeadersArgs(authHeader))
		})
	}

	_, err = tab.Navigate(opts.Url, 120*time.Second)
	if err != nil {
		return nil, err
	}

	time.Sleep(opts.Delay)

	var screenshot string

	if opts.Top != 0 || opts.Left != 0 {
		screenshot, err = tab.CaptureScreenshot(chrome.ScreenshotOpts{Width: s.Width, Height: s.Height}, 120*time.Second)
	} else {
		screenshot, err = tab.CaptureScreenshot(chrome.ScreenshotOpts{Width: opts.Width, Height: opts.Height}, 120*time.Second)
	}
	if err != nil {
		return nil, err
	}

	rawImage, err := base64.StdEncoding.DecodeString(screenshot[strings.Index(screenshot, ",")+1:])
	if err != nil {
		return nil, fmt.Errorf("error: cannot decode base64 image: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(rawImage))
	if err != nil {
		return nil, fmt.Errorf("error: bad png: %v", err)
	}

	//clippedImage := image.NewRGBA(image.Rect(572, 40, 1372, 640))
	if opts.Top != 0 || opts.Left != 0 {
		clippedImage := image.NewRGBA(image.Rect(opts.Left, opts.Top, opts.Left+opts.Width, opts.Top+opts.Height))
		draw.Draw(clippedImage, clippedImage.Rect, img, image.Point{X: opts.Top, Y: opts.Left}, draw.Src)

		img = clippedImage
	}

	var buf bytes.Buffer

	err = png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	err = s.Browser.CloseTab(tab, 120*time.Second)
	if err != nil {
		return nil, err
	}

	log.Println(s.getLogLine(opts))

	return buf.Bytes(), nil
}

func (s *Screenshot) getLogLine(opts *Opts) string {
	var builder strings.Builder

	builder.WriteString("captured screenshot for :: ")
	builder.WriteString(fmt.Sprintf("url: %s ", opts.Url))

	if opts.FullPage {
		builder.WriteString(fmt.Sprintf("[full page screenshot] "))
	}

	if opts.Top > 0 || opts.Left > 0 {
		builder.WriteString("[clipped screenshot] ")
		builder.WriteString(fmt.Sprintf("top: %d ", opts.Top))
		builder.WriteString(fmt.Sprintf("left: %d ", opts.Left))
	}

	builder.WriteString(fmt.Sprintf("width: %d ", opts.Width))
	builder.WriteString(fmt.Sprintf("height: %d ", opts.Height))

	if opts.Delay > 0 {
		builder.WriteString(fmt.Sprintf("delay: %s ", opts.Delay))
	}

	if opts.UserAgent != "" {
		builder.WriteString(fmt.Sprintf("useragent: %s\n", opts.UserAgent))
	}

	return builder.String()
}
