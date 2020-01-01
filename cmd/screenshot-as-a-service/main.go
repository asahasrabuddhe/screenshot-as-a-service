package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"go.ajitem.com/screenshot"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var Version string

func main() {
	app := cli.NewApp()

	app.Name = "screenshot-as-a-service"
	app.Version = Version
	app.Authors = []*cli.Author{
		{
			Name:  "Ajitem Sahasrabuddhe",
			Email: "ajitem.s@outlook.com",
		},
	}

	app.Flags = []cli.Flag{
		&cli.PathFlag{
			Name:     "chromium-path",
			Aliases:  []string{"c"},
			Required: true,
			Usage:    "Path to the chromium binary",
		},
		&cli.IntFlag{
			Name:    "chromium-port",
			Aliases: []string{"d"},
			Value:   4040,
			Usage:   "Port where the headless chromium process will listen",
		},
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"a"},
			Value:   "",
			Usage:   "Broadcast address for the server",
		},
		&cli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Value:   3030,
			Usage:   "Broadcast port for the server",
		},
	}

	app.Action = Action

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Action(context *cli.Context) error {
	path := context.Path("c")
	devToolsPort := context.Int("d")
	address := context.String("a")
	port := context.Int("p")

	s := screenshot.NewScreenshot(path, devToolsPort)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		log.Println("attempting to shutdown gracefully...")

		err := s.Terminate()
		if err != nil {
			log.Fatal(err)
		}

		os.Exit(1)
	}()

	http.Handle("/", s)
	log.Printf("screenshot-as-a-service version %s\nlistening on %s:%d\n", Version, address, port)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
}
