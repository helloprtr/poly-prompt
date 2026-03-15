package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type serverCommandOptions struct {
	addr string
}

type serverExecRequest struct {
	Mode      string `json:"mode"`
	Message   string `json:"message"`
	Target    string `json:"target"`
	DryRun    bool   `json:"dry_run"`
	NoContext bool   `json:"no_context"`
	JSON      bool   `json:"json"`
}

type serverExecResponse struct {
	Target       string `json:"target"`
	TargetSource string `json:"target_source,omitempty"`
	TargetReason string `json:"target_reason,omitempty"`
	FinalPrompt  string `json:"final_prompt,omitempty"`
	Output       string `json:"output,omitempty"`
}

func (a *App) runServer(ctx context.Context, args []string) error {
	command, err := parseServerCommand(args)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    command.addr,
		Handler: a.serverMux(ctx),
	}
	_, _ = fmt.Fprintf(a.stdout, "prtr server listening on %s\n", command.addr)
	return server.ListenAndServe()
}

func parseServerCommand(args []string) (serverCommandOptions, error) {
	command := serverCommandOptions{addr: "127.0.0.1:8787"}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--addr":
			i++
			if i >= len(args) {
				return serverCommandOptions{}, usageError{message: "--addr requires a value", helpText: serverHelpText()}
			}
			command.addr = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--addr="):
			command.addr = strings.TrimSpace(strings.TrimPrefix(arg, "--addr="))
		case strings.HasPrefix(arg, "-"):
			return serverCommandOptions{}, usageError{message: fmt.Sprintf("unknown server flag %q", arg), helpText: serverHelpText()}
		default:
			return serverCommandOptions{}, usageError{message: fmt.Sprintf("unexpected server argument %q", arg), helpText: serverHelpText()}
		}
	}
	return command, nil
}

func (a *App) serverMux(ctx context.Context) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req serverExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resolved, err := a.prepareExecRun(ctx, execCommandOptions{
			mode:      blankDefault(req.Mode, "ask"),
			app:       req.Target,
			dryRun:    req.DryRun,
			noContext: req.NoContext,
			json:      req.JSON,
			prompt:    []string{req.Message},
		}, bytes.NewBuffer(nil), false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := serverExecResponse{
			Target:       resolved.targetName,
			TargetSource: resolved.targetSource,
			TargetReason: resolved.targetReason,
			FinalPrompt:  resolved.finalPrompt,
		}

		if !req.DryRun {
			result, err := a.resolveHeadlessRunner().Run(ctx, HeadlessRequest{
				Target:     resolved.targetName,
				Prompt:     resolved.finalPrompt,
				JSONOutput: req.JSON,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			response.Output = strings.TrimSpace(result.Stdout)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	return mux
}

func serverHelpText() string {
	return strings.Join([]string{
		"Start the alpha HTTP orchestration server.",
		"",
		"`prtr server` exposes `/healthz` and `/exec` so external tools can",
		"compile and run headless requests through the same prompt pipeline.",
		"",
		"Usage:",
		"  prtr server [--addr 127.0.0.1:8787]",
	}, "\n")
}
