package pipeline

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockStage struct {
	name    string
	runFunc func(c *Context) error
}

func (m *mockStage) Name() string { return m.name }
func (m *mockStage) Execute(c *Context) error {
	if m.runFunc != nil {
		return m.runFunc(c)
	}
	return nil
}

func TestPipeline_SequentialExecution(t *testing.T) {
	var executed []string

	s1 := &mockStage{name: "stage1", runFunc: func(c *Context) error {
		executed = append(executed, "stage1")
		return nil
	}}
	s2 := &mockStage{name: "stage2", runFunc: func(c *Context) error {
		executed = append(executed, "stage2")
		return nil
	}}

	p := New(s1, s2)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := NewContext(rec, req)

	if err := p.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(executed) != 2 || executed[0] != "stage1" || executed[1] != "stage2" {
		t.Fatalf("expected [stage1, stage2], got: %v", executed)
	}
}

func TestPipeline_ShortCircuitOnHandled(t *testing.T) {
	var executed []string

	s1 := &mockStage{name: "stage1", runFunc: func(c *Context) error {
		executed = append(executed, "stage1")
		c.MarkHandled()
		return nil
	}}
	s2 := &mockStage{name: "stage2", runFunc: func(c *Context) error {
		executed = append(executed, "stage2")
		return nil
	}}

	p := New(s1, s2)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := NewContext(rec, req)

	if err := p.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(executed) != 1 || executed[0] != "stage1" {
		t.Fatalf("expected only stage1 to execute, got: %v", executed)
	}
}

func TestPipeline_HaltOnError(t *testing.T) {
	expectedErr := errors.New("pipeline failed")
	s1 := &mockStage{name: "stage1", runFunc: func(c *Context) error {
		return expectedErr
	}}
	s2 := &mockStage{name: "stage2", runFunc: func(c *Context) error {
		t.Fatal("stage2 should not have run")
		return nil
	}}

	p := New(s1, s2)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := NewContext(rec, req)

	err := p.Execute(ctx)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestResponseRecorder_Tracking(t *testing.T) {
	baseRec := httptest.NewRecorder()
	rec := &ResponseRecorder{ResponseWriter: baseRec, StatusCode: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)
	if rec.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.StatusCode)
	}

	data := []byte("hello world")
	n, err := rec.Write(data)
	if err != nil || n != len(data) {
		t.Fatalf("write failed: %v", err)
	}
	if rec.BytesWritten != int64(len(data)) {
		t.Errorf("expected %d bytes, got %d", len(data), rec.BytesWritten)
	}
}
