package pixelgl

import (
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// Due to the limitations of OpenGL and operating systems, all OpenGL related calls must be done from the main thread.

var callQueue = make(chan func(), 32)

func init() {
	runtime.LockOSThread()
}

// Run is essentialy the "main" function of the pixelgl package.
// Run this function from the main function (because that's guaranteed to run in the main thread).
//
// This function reserves the main thread for the OpenGL stuff and runs a supplied run function in a
// separate goroutine.
//
// Run returns when the provided run function finishes.
func Run(run func()) {
	done := make(chan struct{})

	go func() {
		run()
		close(done)
	}()

loop:
	for {
		select {
		case f := <-callQueue:
			f()
		case <-done:
			break loop
		}
	}
}

// Init initializes OpenGL by loading the function pointers from the active OpenGL context.
// This function must be manually run inside the main thread (Do, DoErr, DoVal, etc.).
//
// It must be called under the presence of an active OpenGL context, e.g., always after calling window.MakeContextCurrent().
// Also, always call this function when switching contexts.
func Init() {
	err := gl.Init()
	if err != nil {
		panic(err)
	}
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

// DoNoBlock executes a function inside the main OpenGL thread.
// DoNoBlock does not wait until the function finishes.
func DoNoBlock(f func()) {
	callQueue <- f
}

// Do executes a function inside the main OpenGL thread.
// Do blocks until the function finishes.
//
// All OpenGL calls must be done in the dedicated thread.
func Do(f func()) {
	done := make(chan bool)
	callQueue <- func() {
		f()
		done <- true
	}
	<-done
}

// DoErr executes a function inside the main OpenGL thread and returns an error to the called.
// DoErr blocks until the function finishes.
//
// All OpenGL calls must be done in the dedicated thread.
func DoErr(f func() error) error {
	err := make(chan error)
	callQueue <- func() {
		err <- f()
	}
	return <-err
}

// DoVal executes a function inside the main OpenGL thread and returns a value to the caller.
// DoVal blocks until the function finishes.
//
// All OpenGL calls must be done in the main thread.
func DoVal(f func() interface{}) interface{} {
	val := make(chan interface{})
	callQueue <- func() {
		val <- f()
	}
	return <-val
}

// DoGLErr is same as Do, but also return an error generated by OpenGL.
func DoGLErr(f func()) (gl error) {
	glerr := make(chan error)
	callQueue <- func() {
		getLastGLErr() // swallow
		f()
		glerr <- getLastGLErr()
	}
	return <-glerr
}

// DoErrGLErr is same as DoErr, but also returns an error generated by OpenGL.
func DoErrGLErr(f func() error) (_, gl error) {
	err := make(chan error)
	glerr := make(chan error)
	callQueue <- func() {
		getLastGLErr() // swallow
		err <- f()
		glerr <- getLastGLErr()
	}
	return <-err, <-glerr
}

// DoValGLErr is same as DoVal, but also returns an error generated by OpenGL.
func DoValGLErr(f func() interface{}) (_ interface{}, gl error) {
	val := make(chan interface{})
	glerr := make(chan error)
	callQueue <- func() {
		getLastGLErr() // swallow
		val <- f()
		glerr <- getLastGLErr()
	}
	return <-val, <-glerr
}

// GLError represents an error code generated by OpenGL.
type GLError uint32

// Error returns a human-readable textual representation of an OpenGL error.
func (err GLError) Error() string {
	if desc, ok := glErrors[uint32(err)]; ok {
		return fmt.Sprintf("OpenGL error: %s", desc)
	}
	return fmt.Sprintf("OpenGL error: unknown error")
}

var glErrors = map[uint32]string{
	gl.INVALID_ENUM:                  "invalid enum",
	gl.INVALID_VALUE:                 "invalid value",
	gl.INVALID_OPERATION:             "invalid operation",
	gl.STACK_OVERFLOW:                "stack overflow",
	gl.STACK_UNDERFLOW:               "stack underflow",
	gl.OUT_OF_MEMORY:                 "out of memory",
	gl.INVALID_FRAMEBUFFER_OPERATION: "invalid framebuffer operation",
	gl.CONTEXT_LOST:                  "context lost",
}

// getLastGLErr returns (and consumes) the last error generated by OpenGL.
// Don't use outside DoGLErr, DoErrGLErr and DoValGLErr.
func getLastGLErr() error {
	err := uint32(gl.NO_ERROR)
	for e := gl.GetError(); e != gl.NO_ERROR; e = gl.GetError() {
		err = e
	}
	if err != gl.NO_ERROR {
		return GLError(err)
	}
	return nil
}
