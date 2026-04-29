package process

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	core "dappco.re/go"
)

type Stream string

const (
	StreamStdout Stream = "stdout"
	StreamStderr Stream = "stderr"
)

type ActionProcessStarted struct {
	ID      string
	Command string
	Args    []string
	Dir     string
	PID     int
}

type ActionProcessOutput struct {
	ID     string
	Line   string
	Stream Stream
}

type ActionProcessExited struct {
	ID       string
	ExitCode int
	Duration time.Duration
	Error    error
}

type ActionProcessKilled struct {
	ID     string
	Signal string
}

type RunOptions struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
}

type Status string

const (
	StatusRunning Status = "running"
	StatusExited  Status = "exited"
	StatusKilled  Status = "killed"
)

type Info struct {
	ID        string
	Command   string
	Args      []string
	Dir       string
	Status    Status
	PID       int
	ExitCode  int
	StartedAt time.Time
	Duration  time.Duration
}

type captureBuffer struct {
	mu  sync.Mutex
	buf captureBytes
}

type captureBytes []byte

func (b captureBytes) Len() int { return len(b) }

func (b *captureBuffer) Write(p []byte) (
	int,
	error,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *captureBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

type Process struct {
	ID        string
	Command   string
	Args      []string
	Dir       string
	StartedAt time.Time
	ExitCode  int

	cmd    *core.Cmd
	stdin  io.WriteCloser
	output captureBuffer
	done   chan struct{}

	mu       sync.RWMutex
	status   Status
	pid      int
	duration time.Duration
}

func (p *Process) Info() Info {
	p.mu.RLock()
	defer p.mu.RUnlock()
	status := p.status
	duration := p.duration
	if status == StatusRunning {
		duration = time.Since(p.StartedAt)
	}
	return Info{
		ID:        p.ID,
		Command:   p.Command,
		Args:      append([]string(nil), p.Args...),
		Dir:       p.Dir,
		Status:    status,
		PID:       p.pid,
		ExitCode:  p.ExitCode,
		StartedAt: p.StartedAt,
		Duration:  duration,
	}
}

func (p *Process) Done() <-chan struct{} {
	if p.done == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return p.done
}

func (p *Process) Output() string { return p.output.String() }

func (p *Process) Shutdown() (
	_ error, // result
) {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if err := p.cmd.Process.Signal(syscall.SIGINT); err != nil {
		return p.cmd.Process.Kill()
	}
	select {
	case <-p.Done():
		return nil
	case <-time.After(500 * time.Millisecond):
		return p.cmd.Process.Kill()
	}
}

func (p *Process) SendInput(
	input string,
) (
	_ error, // result
) {
	if p.stdin == nil {
		return core.NewError("stdin unavailable")
	}
	_, err := io.WriteString(p.stdin, input)
	return err
}

type Service struct {
	mu      sync.RWMutex
	procs   map[string]*Process
	counter atomic.Uint64
}

type Options struct{}

func NewService(Options) func(*core.Core) (
	any,
	error,
) {
	return func(*core.Core) (any, error) {
		return &Service{}, nil
	}
}

func (s *Service) OnStartup(context.Context) error { return nil }

func (s *Service) OnShutdown(
	context.Context,
) (
	_ error, // result
) {
	for _, proc := range s.List() {
		if err := proc.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) StartWithOptions(ctx context.Context, opts RunOptions) (
	*Process,
	error,
) {
	if opts.Command == "" {
		return nil, core.NewError("command cannot be empty")
	}
	cmd := newCommand(ctx, opts.Command, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(core.Environ(), opts.Env...)
	}

	proc := &Process{
		ID:        core.Sprintf("proc-%d", s.counter.Add(1)),
		Command:   opts.Command,
		Args:      append([]string(nil), opts.Args...),
		Dir:       opts.Dir,
		StartedAt: time.Now(),
		done:      make(chan struct{}),
		status:    StatusRunning,
		cmd:       cmd,
	}
	cmd.Stdout = &proc.output
	cmd.Stderr = &proc.output
	stdin, err := cmd.StdinPipe()
	if err == nil {
		proc.stdin = stdin
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	proc.pid = cmd.Process.Pid

	s.ensure()
	s.mu.Lock()
	s.procs[proc.ID] = proc
	s.mu.Unlock()

	go proc.wait()
	return proc, nil
}

func (p *Process) wait() {
	err := p.cmd.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.duration = time.Since(p.StartedAt)
	if err != nil {
		if exitErr, ok := err.(interface{ ExitCode() int }); ok {
			p.ExitCode = exitErr.ExitCode()
		} else {
			p.ExitCode = -1
		}
	} else {
		p.ExitCode = 0
	}
	if p.status == StatusRunning {
		p.status = StatusExited
	}
	close(p.done)
}

func (s *Service) Get(id string) (
	*Process,
	error,
) {
	s.ensure()
	s.mu.RLock()
	defer s.mu.RUnlock()
	proc, ok := s.procs[id]
	if !ok {
		return nil, core.NewError("process not found")
	}
	return proc, nil
}

func newCommand(ctx context.Context, command string, args ...string) *core.Cmd {
	if core.Contains(command, "/") {
		return &core.Cmd{Path: command, Args: append([]string{command}, args...)}
	}
	script := command
	for _, arg := range args {
		script = core.Concat(script, " ", shellQuote(arg))
	}
	cmd := &core.Cmd{Path: "/bin/sh", Args: []string{"sh", "-c", script}}
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			if killErr := cmd.Process.Kill(); killErr != nil {
				core.Error("process kill failed", "err", killErr)
			}
		}
	}()
	return cmd
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + core.Replace(value, "'", "'\"'\"'") + "'"
}

func (s *Service) List() []*Process {
	s.ensure()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Process, 0, len(s.procs))
	for _, proc := range s.procs {
		out = append(out, proc)
	}
	return out
}

func (s *Service) Running() []*Process {
	all := s.List()
	out := make([]*Process, 0, len(all))
	for _, proc := range all {
		if proc.Info().Status == StatusRunning {
			out = append(out, proc)
		}
	}
	return out
}

func (s *Service) Kill(
	id string,
) (
	_ error, // result
) {
	proc, err := s.Get(id)
	if err != nil {
		return err
	}
	if proc.cmd == nil || proc.cmd.Process == nil {
		return nil
	}
	if err := proc.cmd.Process.Kill(); err != nil {
		return err
	}
	proc.mu.Lock()
	proc.status = StatusKilled
	proc.mu.Unlock()
	return nil
}

func (s *Service) Output(id string) (
	string,
	error,
) {
	proc, err := s.Get(id)
	if err != nil {
		return "", err
	}
	return proc.Output(), nil
}

func (s *Service) ensure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.procs == nil {
		s.procs = make(map[string]*Process)
	}
}
