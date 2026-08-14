package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"golang.org/x/sync/errgroup"
)

// Stream delivers lines and errors from a running docker logs tail.
type Stream struct {
	Lines  <-chan string
	Errors <-chan error
	Stop   func()
}

// Tail runs docker logs -f for the named container and returns a stream of
// stdout lines and stderr errors. Call Stop to terminate the tail.
func Tail(containerName string) (*Stream, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", "--tail", "100", containerName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start docker logs: %w", err)
	}

	lines := make(chan string)
	errs := make(chan error)
	var eg errgroup.Group

	eg.Go(func() error { return streamLines(ctx, stdout, lines) })
	eg.Go(func() error { return streamErrors(ctx, stderr, errs) })

	go func() {
		_ = cmd.Wait()
		cancel()
		_ = eg.Wait()
		close(lines)
		close(errs)
	}()

	return &Stream{
		Lines:  lines,
		Errors: errs,
		Stop:   cancel,
	}, nil
}

func streamLines(ctx context.Context, r io.Reader, out chan<- string) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case out <- scanner.Text():
		case <-ctx.Done():
			return nil
		}
	}
	return scanner.Err()
}

func streamErrors(ctx context.Context, r io.Reader, out chan<- error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case out <- errors.New(scanner.Text()):
		case <-ctx.Done():
			return nil
		}
	}
	return scanner.Err()
}
