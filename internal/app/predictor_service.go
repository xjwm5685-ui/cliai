package app

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sanqiu/cliai/internal/config"
	"github.com/sanqiu/cliai/internal/feedback"
	"github.com/sanqiu/cliai/internal/history"
	"github.com/sanqiu/cliai/internal/predict"
	"github.com/sanqiu/cliai/internal/project"
)

const predictorRefreshInterval = 2 * time.Second

type predictorServiceRequest struct {
	Input string `json:"input"`
	CWD   string `json:"cwd,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type predictorServiceResponse struct {
	Suggestions []predict.Candidate `json:"suggestions,omitempty"`
	Error       string              `json:"error,omitempty"`
}

type predictorQueryService interface {
	Query(request predictorServiceRequest) ([]predict.Candidate, error)
}

type predictorService struct {
	mu            sync.Mutex
	cfg           *config.Config
	engine        *predict.Predictor
	cachePath     string
	feedbackPath  string
	history       []history.Entry
	feedbackItems []feedback.Entry
	defaultLimit  int
	shell         string
	lastRefresh   time.Time
}

func runPredictor(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "serve" {
		fmt.Fprintln(stderr, "usage: cliai predictor serve [--limit 8] [--shell powershell]")
		return 1
	}

	fs := flag.NewFlagSet("predictor serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	limit := fs.Int("limit", 8, "default number of suggestions to return")
	shell := fs.String("shell", "powershell", "shell type")
	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	service, err := newPredictorService(*limit, *shell)
	if err != nil {
		fmt.Fprintf(stderr, "start predictor service: %v\n", err)
		return 1
	}

	return runPredictorStream(service, os.Stdin, stdout, stderr)
}

func runPredictorStream(service predictorQueryService, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	writer := bufio.NewWriter(stdout)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var request predictorServiceRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			_ = encoder.Encode(predictorServiceResponse{Error: fmt.Sprintf("decode request: %v", err)})
			_ = writer.Flush()
			continue
		}

		suggestions, queryErr := service.Query(request)
		response := predictorServiceResponse{Suggestions: suggestions}
		if queryErr != nil {
			response.Error = queryErr.Error()
		}

		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "write predictor response: %v\n", err)
			return 1
		}
		if err := writer.Flush(); err != nil {
			fmt.Fprintf(stderr, "flush predictor response: %v\n", err)
			return 1
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "read predictor request: %v\n", err)
		return 1
	}
	return 0
}

func newPredictorService(limit int, shell string) (*predictorService, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	cachePath, err := config.HistoryCachePath()
	if err != nil {
		return nil, err
	}

	feedbackPath, err := config.FeedbackPath()
	if err != nil {
		return nil, err
	}

	service := &predictorService{
		cfg:          cfg,
		engine:       predict.New(),
		cachePath:    cachePath,
		feedbackPath: feedbackPath,
		defaultLimit: limit,
		shell:        normalizeShell(shell),
	}

	if err := service.refreshLocked(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *predictorService) Query(request predictorServiceRequest) ([]predict.Candidate, error) {
	input := strings.TrimSpace(request.Input)
	if input == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.lastRefresh) >= predictorRefreshInterval {
		if err := s.refreshLocked(); err != nil {
			return nil, err
		}
	}

	cwd := strings.TrimSpace(request.CWD)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	projectCtx, err := project.Detect(cwd)
	if err != nil {
		return nil, fmt.Errorf("detect project context: %w", err)
	}

	limit := request.Limit
	if limit <= 0 {
		limit = s.defaultLimit
	}

	candidates := s.engine.Predict(predict.Request{
		Query:           input,
		CWD:             cwd,
		Shell:           s.shell,
		Limit:           limit,
		Project:         projectCtx,
		FeedbackBonuses: feedback.CommandBonuses(input, s.feedbackItems),
	}, s.history)
	return candidates, nil
}

func (s *predictorService) refreshLocked() error {
	cfg, err := config.Load()
	if err == nil {
		s.cfg = cfg
	}

	cached, err := history.LoadCache(s.cachePath)
	if err != nil {
		return fmt.Errorf("load cached history: %w", err)
	}

	var live []history.Entry
	if s.cfg != nil && s.cfg.HistoryPath != "" {
		live, err = history.Import(s.cfg.HistoryPath, s.cfg.Shell, s.cfg.Local.MaxHistory)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read powershell history: %w", err)
		}
	}

	feedbackItems, err := feedback.Load(s.feedbackPath)
	if err != nil {
		return fmt.Errorf("load feedback: %w", err)
	}

	s.history = history.Merge(cached, live)
	s.feedbackItems = feedbackItems
	s.lastRefresh = time.Now()
	return nil
}
