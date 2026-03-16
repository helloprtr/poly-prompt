package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/deep"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

// HeadlessRequest is the input to a headless AI runner.
type HeadlessRequest struct {
	Target     string
	Prompt     string
	JSONOutput bool
}

// HeadlessResult is the output from a headless AI runner.
type HeadlessResult struct {
	Stdout string
}

// HeadlessRunner executes a prompt against an AI target without a UI.
type HeadlessRunner interface {
	Run(ctx context.Context, req HeadlessRequest) (HeadlessResult, error)
}

// noopHeadlessRunner is a placeholder that always returns an error.
type noopHeadlessRunner struct{}

func (noopHeadlessRunner) Run(_ context.Context, _ HeadlessRequest) (HeadlessResult, error) {
	return HeadlessResult{}, errors.New("headless runner is not configured")
}

func (a *App) resolveHeadlessRunner() HeadlessRunner {
	return noopHeadlessRunner{}
}

// execCommandOptions are the parsed fields from a server /exec request.
type execCommandOptions struct {
	mode      string
	app       string
	dryRun    bool
	noContext  bool
	json      bool
	prompt    []string
}

// serverExecRequest is the JSON body for POST /exec.
type serverExecRequest struct {
	Message   string `json:"message"`
	Target    string `json:"target"`
	Mode      string `json:"mode"`
	DryRun    bool   `json:"dry_run"`
	NoContext  bool   `json:"no_context"`
	JSON      bool   `json:"json"`
}

// serverExecResponse is the JSON body returned by POST /exec.
type serverExecResponse struct {
	Target       string `json:"target"`
	TargetSource string `json:"target_source"`
	TargetReason string `json:"target_reason"`
	FinalPrompt  string `json:"final_prompt"`
	Output       string `json:"output,omitempty"`
}

// prepareExecRun resolves config, translates the prompt, and renders the
// template — mirroring prepareRun but driven by execCommandOptions.
func (a *App) prepareExecRun(ctx context.Context, opts execCommandOptions, _ *bytes.Buffer, noContext bool) (resolvedRun, error) {
	text := strings.Join(opts.prompt, " ")

	repoSuffix := ""
	if !noContext && !opts.noContext {
		if a.repoContext != nil {
			summary, err := a.repoContext.Collect(ctx)
			if err == nil {
				repoSuffix = formatRepoContext(summary)
			}
		}
	}

	protectedTerms, protectedSuffix := a.resolveLearnedTerms(opts.noContext)

	runOpts := runOptions{
		target:               strings.TrimSpace(opts.app),
		noCopy:               true,
		surfaceMode:          blankDefault(opts.mode, "ask"),
		surfaceInput:         "prompt",
		surfaceDelivery:      "dry-run",
		promptSuffix:         joinPromptSections(repoSuffix, protectedSuffix),
		protectedTerms:       protectedTerms,
		preferTargetTemplate: true,
		jsonOutput:           opts.json,
	}

	return a.prepareRun(ctx, runOpts, text, blankDefault(opts.mode, "ask"))
}

// serverCommandOptions holds the parsed server command arguments.
type serverCommandOptions struct {
	addr string
}

func parseServerCommand(args []string) (serverCommandOptions, error) {
	cmd := serverCommandOptions{addr: ":8080"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--addr" || arg == "-addr":
			i++
			if i >= len(args) {
				return serverCommandOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			cmd.addr = args[i]
		case strings.HasPrefix(arg, "--addr="):
			cmd.addr = strings.TrimPrefix(arg, "--addr=")
		default:
			return serverCommandOptions{}, fmt.Errorf("unknown server flag %q", arg)
		}
	}
	return cmd, nil
}

// serverDeepExecRequest is the JSON body for POST /exec/deep.
type serverDeepExecRequest struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	RepoRoot string `json:"repo_root"`
	DryRun   bool   `json:"dry_run"`
	// Approved must be true for the caller to acknowledge the bundle is cleared
	// for delivery. Workers have no mutation rights; the gate lives here.
	Approved bool `json:"approved"`
}

// serverDeepExecResponse is the JSON body returned by POST /exec/deep.
type serverDeepExecResponse struct {
	RunID            string   `json:"run_id"`
	Status           string   `json:"status"`
	ArtifactRoot     string   `json:"artifact_root"`
	BundlePath       string   `json:"bundle_path"`
	BundleSummary    string   `json:"bundle_summary"`
	WarningCount     int      `json:"warning_count"`
	Warnings         []string `json:"warnings,omitempty"`
	ApprovalRequired bool     `json:"approval_required,omitempty"`
	Approved         bool     `json:"approved"`
}

// serverMux builds the HTTP handler used by the server.
func (a *App) serverMux(ctx context.Context) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
		reqCtx := r.Context()

		var req serverExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resolved, err := a.prepareExecRun(reqCtx, execCommandOptions{
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
			TargetReason: resolved.targetSource,
			FinalPrompt:  resolved.finalPrompt,
		}

		if !req.DryRun {
			result, err := a.resolveHeadlessRunner().Run(reqCtx, HeadlessRequest{
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

	// POST /exec/deep — run the deep patch engine and return a PatchBundle.
	//
	// Approval gate: workers have no mutation rights (shell, git, filesystem).
	// The bundle is returned for review. Set "approved": true in the request to
	// acknowledge the bundle and signal it is cleared for delivery by the caller.
	// The server itself does not launch apps or write to the repository.
	mux.HandleFunc("/exec/deep", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		reqCtx := r.Context()

		var req serverDeepExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Source) == "" {
			http.Error(w, "source is required", http.StatusBadRequest)
			return
		}

		repoRoot := strings.TrimSpace(req.RepoRoot)
		repoSummary := repoctx.Summary{}
		if a.repoContext != nil {
			if summary, err := a.repoContext.Collect(reqCtx); err == nil {
				repoSummary = summary
			}
		}
		protectedTerms, _ := a.resolveLearnedTerms(false)

		result, err := deep.ExecutePatchRun(reqCtx, deep.Options{
			Action:         "patch",
			Source:         req.Source,
			SourceKind:     "server",
			TargetApp:      strings.TrimSpace(req.Target),
			RepoRoot:       repoRoot,
			ProtectedTerms: protectedTerms,
			RepoSummary:    repoSummary,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := serverDeepExecResponse{
			RunID:            result.Run.ID,
			Status:           string(result.Run.Status),
			ArtifactRoot:     result.Run.ArtifactRoot,
			BundlePath:       result.Run.ArtifactRoot + "/result/patch_bundle.json",
			BundleSummary:    result.Bundle.Summary,
			WarningCount:     result.Run.WarningCount,
			Warnings:         result.Bundle.Warnings,
			ApprovalRequired: !req.Approved,
			Approved:         req.Approved,
		}

		// If the caller has not yet approved, return 202 so they can review the bundle.
		if !req.Approved {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	_ = ctx
	return mux
}

// runServer starts the HTTP server and blocks until ctx is cancelled or a
// fatal listen error occurs. On cancellation it performs a graceful shutdown.
func (a *App) runServer(ctx context.Context, args []string) error {
	command, err := parseServerCommand(args)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:         command.addr,
		Handler:      a.serverMux(ctx),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		_, _ = fmt.Fprintf(a.stdout, "prtr server listening on %s\n", command.addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
