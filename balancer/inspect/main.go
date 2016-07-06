package main

/*
#include <stdlib.h>
#include <dlfcn.h>
#cgo LDFLAGS: -L. -ldl

void *load(char *path) {
  void *lib = dlopen(path, RTLD_LAZY);
  free(path);
  return lib;
}

void *get_func(void *mod, char *name) {
  void *fn = dlsym(mod, name);
  free(name);
  return fn;
}

void call_func(void *raw) {
  void (*fn)(void) = raw;
  fn();
}

*/
import "C"

import (
	"fmt"
	"os"
	"unsafe"
)

type Module struct {
	module  unsafe.Pointer
	test_fn unsafe.Pointer
}

func NewModule(name string) (*Module, error) {
	module := C.load(C.CString(name))
	if module == nil {
		return nil, fmt.Errorf("could not open %v", name)
	}

	test_fn := C.get_func(module, C.CString("test"))
	if test_fn == nil {
		// TODO: release module
		return nil, fmt.Errorf("could not find test function")
	}

	return &Module{module: module, test_fn: test_fn}, nil
}

func (m *Module) CallTest() {
	C.call_func(m.test_fn)
}

func main() {
	fmt.Printf("Load module dynamically\n")
	mod, err := NewModule(os.Args[1])
	if err != nil {
		fmt.Printf("Failed with: %v\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Calling test() in loaded module:\n")
	mod.CallTest()
}
