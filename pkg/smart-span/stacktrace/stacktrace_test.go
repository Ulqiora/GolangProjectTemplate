package stacktrace

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestFormatStackIncludesLastFrame(t *testing.T) {
	t.Parallel()

	stack := &Stack{
		pcs: []uintptr{1},
		frames: runtime.CallersFrames([]uintptr{
			mustFuncPC(t, helperStacktraceLeaf),
		}),
	}

	formatted := NewStackFormatter().FormatStack(stack).String()
	if !strings.Contains(formatted, "helperStacktraceLeaf") {
		t.Fatalf("expected formatted stack to include last frame, got %q", formatted)
	}
}

func TestTakeCallerFunctionInfoAlwaysReturnsFunctionName(t *testing.T) {
	t.Parallel()

	frame := helperTakeCallerFunctionInfo()
	if frame.Name == "" {
		t.Fatal("expected caller function info to include function name")
	}
	if frame.File == "" {
		t.Fatal("expected caller function info to include file")
	}
}

func helperStacktraceLeaf() {}

func helperTakeCallerFunctionInfo() FunctionFrame {
	return TakeCallerFunctionInfo()
}

func TestCallerFunctionName_ShortAndNonEmpty(t *testing.T) {
	t.Parallel()

	got := helperCallerFunctionName()
	if got == "" {
		t.Fatal("expected caller function name to be non-empty")
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("expected single-line function name, got %q", got)
	}
	if !strings.Contains(got, "helperCallerFunctionName") {
		t.Fatalf("expected function name to include helperCallerFunctionName, got %q", got)
	}
}

func helperCallerFunctionName() string {
	return CallerFunctionName(0)
}

func mustFuncPC(t *testing.T, fn func()) uintptr {
	t.Helper()

	pc := reflect.ValueOf(fn).Pointer()
	if pc == 0 {
		t.Fatal("expected function PC")
	}
	return pc
}
