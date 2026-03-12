package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/helloprtr/poly-prompt/internal/app"
	"github.com/helloprtr/poly-prompt/internal/clipboard"
	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/editor"
	"github.com/helloprtr/poly-prompt/internal/input"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

var version = "dev"

func main() {
	stdinPiped, err := input.StdinIsPiped(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to inspect stdin: %v\n", err)
		os.Exit(1)
	}

	application := app.New(app.Dependencies{
		Version: version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		LookupEnv: func(key string) (string, bool) {
			return os.LookupEnv(key)
		},
		ConfigLoader: config.Load,
		ConfigInit:   config.Init,
		Translator: translate.NewDeepLClient(translate.ClientOptions{
			APIKey:  os.Getenv("DEEPL_API_KEY"),
			BaseURL: translate.DefaultBaseURL,
			HTTPClient: &http.Client{
				Timeout: 15 * time.Second,
			},
		}),
		Clipboard: clipboard.New(),
		Editor:    editor.New(os.Stderr),
	})

	if err := application.Execute(context.Background(), os.Args[1:], os.Stdin, stdinPiped); err != nil {
		if errors.Is(err, editor.ErrCanceled) {
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
