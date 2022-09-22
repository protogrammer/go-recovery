package recovery

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"log"
	"os"
	"strings"
)

type CommentType string
type CaptureType any

type StackMetadata any // CommentType | CaptureType | PanicMessage

type StackFrame struct {
	Process  *string
	Metadata []StackMetadata
}

type PanicMessage struct {
	CallStack []StackFrame
	Err       any
}

type PanicHandlerEnum uint

const (
	BLOCK PanicHandlerEnum = iota
	STOP
	RECUR
	RECOVER
)

type HandlePanicFunc func(PanicMessage)
type SpecifyBehaviorFunc func(err any, panicHandler PanicHandlerEnum) PanicHandlerEnum

type Config struct {
	handlePanic     HandlePanicFunc
	specifyBehavior SpecifyBehaviorFunc
}

func CreateConfig(handlePanic HandlePanicFunc, specifyBehavior SpecifyBehaviorFunc) Config {
	return Config{
		handlePanic:     handlePanic,
		specifyBehavior: specifyBehavior,
	}
}

func PanicMessageFromError(err any) PanicMessage {
	if msg, ok := err.(PanicMessage); ok {
		return msg
	}
	return PanicMessage{
		CallStack: nil,
		Err:       err,
	}
}

func (msg PanicMessage) String() string {
	return msg.StringIndent(0)
}

func (msg PanicMessage) Log() {
	log.Print("\n", msg.String())
}

const indentString string = "    "

func (msg PanicMessage) StringIndent(indent uint) string {
	fullIndent := strings.Repeat(indentString, int(indent))
	arr := []string{
		spew.Sprintf("%sPanic: %+v", fullIndent, msg.Err),
		fmt.Sprintf("%sType: %T", fullIndent, msg.Err),
		fmt.Sprint(fullIndent, "Call stack:"),
	}
	if len(msg.CallStack) == 0 {
		arr = append(arr, fullIndent+indentString+"Empty")
	}
	for i := range msg.CallStack {
		frame := msg.CallStack[len(msg.CallStack)-i-1]
		processIndent := " -> "
		if i == 0 {
			processIndent = indentString
		}
		process := "unknown"
		if frame.Process != nil {
			process = *frame.Process
		}
		arr = append(arr, fmt.Sprint(fullIndent, processIndent, process))
		metadataIndent := fullIndent + indentString
		metadataIndent2 := metadataIndent + indentString
		for j := range frame.Metadata {
			metadata := frame.Metadata[len(frame.Metadata)-j-1]
			if msg1, ok := metadata.(PanicMessage); ok {
				arr = append(arr, metadataIndent+"Secondary panic:", msg1.StringIndent(indent+2))
			} else if comment, ok := metadata.(CommentType); ok {
				arr = append(arr, metadataIndent+"Comment: "+string(comment))
			} else if capture, ok := metadata.(CaptureType); ok {
				arr = append(arr,
					metadataIndent+"Capture:",
					spew.Sprintf("%s%#+v", metadataIndent2, capture))
			} else {
				arr = append(arr,
					metadataIndent+"Foreign metadata:",
					spew.Sprintf("%s%#+v", metadataIndent2, metadata))
			}
		}
	}

	return strings.Join(arr, "\n")
}

func (msg *PanicMessage) AddProcess(process string) {
	n := len(msg.CallStack)
	if n == 0 || msg.CallStack[n-1].Process != nil {
		msg.CallStack = append(msg.CallStack, StackFrame{
			Process:  &process,
			Metadata: nil,
		})
		return
	}
	msg.CallStack[n-1].Process = &process
}

func (msg *PanicMessage) AddMetadata(metadata StackMetadata) {
	n := len(msg.CallStack)
	if n == 0 || msg.CallStack[n-1].Process != nil {
		msg.CallStack = append(msg.CallStack, StackFrame{
			Process:  nil,
			Metadata: []StackMetadata{metadata},
		})
		return
	}
	msg.CallStack[n-1].Metadata = append(msg.CallStack[n-1].Metadata, metadata)
}

func (config Config) perform(msg PanicMessage, panicHandler PanicHandlerEnum) {
	if config.specifyBehavior != nil {
		panicHandler = config.specifyBehavior(msg.Err, panicHandler)
	}
	switch panicHandler {
	case BLOCK:
		if config.handlePanic != nil {
			config.handlePanic(msg)
		}
	case STOP:
		if config.handlePanic != nil {
			config.handlePanic(msg)
		}
		os.Exit(-1)
	case RECUR:
		panic(msg)
	}
}

func (config Config) Block(process string) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddProcess(process)

	config.perform(msg, BLOCK)
}

func (config Config) Stop(process string) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddProcess(process)

	config.perform(msg, STOP)
}

func (config Config) Recur(process string) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddProcess(process)

	config.perform(msg, RECUR)
}

func (config Config) Recover(process string) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddProcess(process)

	config.perform(msg, RECOVER)
}

func Comment(values ...any) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddMetadata(CommentType(fmt.Sprint(values...)))

	panic(msg)
}

func Commentln(values ...any) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	s := fmt.Sprintln(values...)
	msg.AddMetadata(CommentType(s[:len(s)-1]))

	panic(msg)
}

func Commentf(format string, values ...any) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddMetadata(CommentType(fmt.Sprintf(format, values...)))

	panic(msg)
}

func CommentResult(f func() string) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddMetadata(CommentType(f()))

	panic(msg)
}

func Capture(value any) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddMetadata(CaptureType(value))

	panic(msg)
}

func CaptureResult(f func() any) {
	err := recover()
	if err == nil {
		return
	}

	msg := PanicMessageFromError(err)
	msg.AddMetadata(CaptureType(f()))

	panic(msg)
}

func Set[T any](ptr *T, val T) {
	err := recover()
	if err == nil {
		return
	}
	*ptr = val
	panic(err)
}

func SetResult[T any](ptr *T, f func() T) {
	err := recover()
	if err == nil {
		return
	}
	*ptr = f()
	panic(err)
}

func internalRecover(err any) {
	err2 := recover()
	if err2 == nil {
		if err == nil {
			return
		}
		panic(err)
	}
	if err == nil {
		panic(err2)
	}

	msg := PanicMessageFromError(err)
	msg2 := PanicMessageFromError(err2)
	msg.AddMetadata(msg2)
	panic(msg)
}

func Defer(f func()) {
	err := recover()
	defer internalRecover(err)
	f()
}

func IfPanic(f func()) {
	err := recover()
	defer internalRecover(err)
	if err != nil {
		f()
	}
}

func IfNotPanic(f func()) {
	err := recover()
	defer internalRecover(err)
	if err == nil {
		f()
	}
}

func Recover() {
	recover()
}
