package stacktrace

import (
	"runtime"
	"strconv"
	"strings"
)

type Depth int

const (
	// First - functionCalled
	First Depth = iota << 6

	// Full - full stacktrace for called function
	Full
)

const (
	Package = iota
	FunctionName
)

var _stackPool = NewPool(func() *Stack {
	return &Stack{
		storage: make([]uintptr, Full),
	}
})

// Stack is a captured stack trace.
type Stack struct {
	pcs    []uintptr // program counters; always a subslice of storage
	frames *runtime.Frames

	// The size of pcs varies depending on requirements:
	// it will be one if the only the first frame was requested,
	// and otherwise it will reflect the depth of the call stack.
	//
	// storage decouples the slice we need (pcs) from the slice we pool.
	// We will always allocate a reasonably large storage, but we'll use
	// only as much of it as we need.
	storage []uintptr
}

type FunctionFrame struct {
	Package string
	File    string
	Line    int
	Name    string
}

func (st *Stack) Free() {
	st.frames = nil
	st.pcs = nil
	_stackPool.Put(st)
}

// Count fetch depth of stacktrace
func (st *Stack) Count() int {
	return len(st.pcs)
}

// Next returns the next frame in the stack trace,
// and a boolean indicating whether there are more after it.
func (st *Stack) Next() (_ runtime.Frame, more bool) {
	return st.frames.Next()
}

func Capture(skip int, depth Depth) *Stack {
	stack := _stackPool.Get()

	switch depth {
	case First:
		stack.pcs = stack.storage[:(First + 2)]
	case Full:
		stack.pcs = stack.storage
	}

	// Unlike other "skip"-based APIs, skip=0 identifies runtime.Callers
	// itself. +2 to skip captureStacktrace and runtime.Callers.
	numFrames := runtime.Callers(
		skip+2,
		stack.pcs,
	)

	// runtime.Callers truncates the recorded stacktrace if there is no
	// room in the provided slice. For the full stack trace, keep expanding
	// storage until there are fewer frames than there is room.
	if depth == Full {
		pcs := stack.pcs
		for numFrames == len(pcs) {
			pcs = make([]uintptr, len(pcs)*2)
			numFrames = runtime.Callers(skip+2, pcs)
		}

		// Discard old storage instead of returning it to the pool.
		// This will adjust the pool size over time if stack traces are
		// consistently very deep.
		stack.storage = pcs
		stack.pcs = pcs[:numFrames]
	} else {
		stack.pcs = stack.pcs[:numFrames]
		//numFrames = runtime.Сфддук
	}

	stack.frames = runtime.CallersFrames(stack.pcs)
	return stack
}

func Take(skip int) string {
	stack := Capture(skip+1, Full)
	defer stack.Free()

	stacktraceInfo := NewStackFormatter().FormatStack(stack)
	return stacktraceInfo.String()
}

func TakeOnceCalledFunction() string {
	stack := Capture(1, First)
	defer stack.Free()

	stacktraceInfo := NewStackFormatter().FormatStack(stack)
	return stacktraceInfo.String()
}

func TakeCallerFunctionInfo() FunctionFrame {
	stack := Capture(1, First)
	defer stack.Free()

	frame, _ := stack.Next()
	functionData := strings.SplitN(frame.Function, ".", 2)

	return FunctionFrame{
		Package: functionData[Package],
		File:    frame.File,
		Line:    frame.Line,
		Name:    functionData[FunctionName],
	}
}

type StackFormatter struct {
	stringBuilder strings.Builder
	firstFrame    bool
}

func NewStackFormatter() *StackFormatter {
	return &StackFormatter{
		stringBuilder: strings.Builder{},
		firstFrame:    true,
	}
}

func (sf *StackFormatter) FormatStack(stack *Stack) *StackFormatter {
	for frame, more := stack.Next(); more; frame, more = stack.Next() {
		sf.FormatFrame(frame)
	}
	return sf
}

func (sf *StackFormatter) String() string {
	return sf.stringBuilder.String()
}

func (sf *StackFormatter) FormatFrame(frame runtime.Frame) {
	if !sf.firstFrame {
		sf.stringBuilder.WriteString("\n")
	}
	sf.firstFrame = false
	sf.stringBuilder.WriteString(frame.Function)
	sf.stringBuilder.WriteString("\n")
	sf.stringBuilder.WriteString("\t")
	sf.stringBuilder.WriteString(frame.File)
	sf.stringBuilder.WriteString(":")
	sf.stringBuilder.WriteString(strconv.Itoa(frame.Line))
}
