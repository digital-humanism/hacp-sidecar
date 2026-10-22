package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const (
	createNewProcessGroup = 0x00000200
	ctrlBreakEvent        = 1
)

type tokenPair struct {
	Env string `json:"env"`
	Tok string `json:"tok"`
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "H03_FAIL: "+format+"\n", args...)
	os.Exit(1)
}

func waitListener(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("listener %s not ready", addr)
}

func listenerGone(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return nil
		}

		_ = c.Close()
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("listener %s still reachable", addr)
}

func readToken(path string) tokenPair {
	f, err := os.Open(path)
	if err != nil {
		fail("open fixture: %v", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	if !s.Scan() {
		fail("fixture empty")
	}

	var tok tokenPair

	if err := json.Unmarshal(s.Bytes(), &tok); err != nil {
		fail("parse fixture: %v", err)
	}

	if tok.Env == "" || tok.Tok == "" {
		fail("fixture missing env/tok")
	}

	return tok
}

func main() {
	if len(os.Args) != 6 {
		fail("usage: helper <sidecar.exe> <fixture> <stdout> <stderr> <provenance>")
	}

	sidecar := os.Args[1]
	fixture := os.Args[2]
	stdoutPath := os.Args[3]
	stderrPath := os.Args[4]
	provPath := os.Args[5]

	stdout, err := os.Create(stdoutPath)
	if err != nil {
		fail("create stdout: %v", err)
	}
	defer stdout.Close()

	stderr, err := os.Create(stderrPath)
	if err != nil {
		fail("create stderr: %v", err)
	}
	defer stderr.Close()

	cmd := exec.Command(sidecar)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Env = append(
		os.Environ(),
		"HACP_TRUST_MODE=test",
		"HACP_CONTROL_MODE=standalone",
		"HACP_SIDECAR_PORT=18082",
		"HACP_UPSTREAM=http://127.0.0.1:18083",
		"HACP_PROVENANCE_FLUSH_PATH="+provPath,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup,
	}

	if err := cmd.Start(); err != nil {
		fail("start sidecar: %v", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("SIDECAR_PID=%d\n", pid)

	waited := false

	defer func() {
		if !waited && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	if err := waitListener("127.0.0.1:18082", 10*time.Second); err != nil {
		fail("%v", err)
	}

	fmt.Println("LISTENER_READY=PASS")

	tok := readToken(fixture)

	req, err := http.NewRequest(
		http.MethodGet,
		"http://127.0.0.1:18082/api/test",
		nil,
	)
	if err != nil {
		fail("new request: %v", err)
	}

	req.Header.Set("X-HACP-Intent-Envelope", tok.Env)
	req.Header.Set("X-HACP-Decision-Token", tok.Tok)

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		fail("authorized request: %v", err)
	}

	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()

	if readErr != nil || closeErr != nil {
		fail("response read=%v close=%v", readErr, closeErr)
	}

	decision := resp.Header.Get("X-HACP-Decision")

	fmt.Printf("AUTHORIZED_HTTP_STATUS=%d\n", resp.StatusCode)
	fmt.Printf("AUTHORIZED_DECISION=%s\n", decision)
	fmt.Printf("AUTHORIZED_BODY_BYTES=%d\n", len(body))

	// Upstream is intentionally absent. Authorization must nevertheless
	// complete before forwarding and therefore return ALLOW semantics.
	if resp.StatusCode != http.StatusBadGateway {
		fail("expected HTTP 502 from absent upstream, got %d", resp.StatusCode)
	}

	if decision != "ALLOW" {
		fail("expected X-HACP-Decision=ALLOW, got %q", decision)
	}

	fmt.Println("AUTHORIZED_PROVENANCE_TRIGGER=PASS")

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	generate := kernel32.NewProc("GenerateConsoleCtrlEvent")

	r1, _, callErr := generate.Call(
		uintptr(ctrlBreakEvent),
		uintptr(pid),
	)

	fmt.Printf("CTRL_BREAK_RESULT=%d\n", r1)

	if r1 == 0 {
		fail("GenerateConsoleCtrlEvent failed: %v", callErr)
	}

	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		waited = true

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				fail(
					"sidecar exited non-zero: %s",
					strconv.Itoa(exitErr.ExitCode()),
				)
			}
			fail("wait sidecar: %v", err)
		}

		fmt.Println("PROCESS_EXIT_CODE=0")

	case <-time.After(10 * time.Second):
		fail("sidecar did not exit after CTRL_BREAK_EVENT")
	}

	if err := listenerGone("127.0.0.1:18082", 5*time.Second); err != nil {
		fail("%v", err)
	}

	fmt.Println("LISTENER_REMOVED=PASS")

	data, err := os.ReadFile(provPath)
	if err != nil {
		fail("read provenance after shutdown: %v", err)
	}

	if len(data) == 0 {
		fail("provenance file empty after graceful shutdown")
	}

	lines := 0
	scanner := bufio.NewScanner(
		bytesReader(data),
	)

	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			lines++
		}
	}

	if err := scanner.Err(); err != nil {
		fail("scan provenance: %v", err)
	}

	if lines < 1 {
		fail("no provenance records persisted")
	}

	fmt.Printf("PROVENANCE_BYTES=%d\n", len(data))
	fmt.Printf("PROVENANCE_RECORDS=%d\n", lines)
	fmt.Println("PROVENANCE_PERSISTED_AFTER_SHUTDOWN=PASS")
	fmt.Println("R4_3_H03=PASS")
}

type sliceReader struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *sliceReader {
	return &sliceReader{data: data}
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
