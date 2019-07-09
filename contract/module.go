package vm

import "C"

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"regexp"
	"strings"
	"unsafe"
)

// const
const (
	JSLibRootName = "jslib/"
)

var (
	pathRe = regexp.MustCompile("^\\.{0,2}/")
)

// Module module structure.
type Module struct {
	id         string
	source     string
	lineOffset int
}

// Modules module maps.
type Modules map[string]*Module

// NewModules create new modules and return it.
func NewModules() Modules {
	return make(Modules, 1)
}

// NewModule create new module and return it.
func NewModule(id, source string, lineOffset int) *Module {
	if !pathRe.MatchString(id) {
		id = fmt.Sprintf("jslib/%s", id)
	}
	id = reformatModuleID(id)
	logger.WithFields(logger.Fields{
		"id": id,
	}).Debug("NewModule id.")
	return &Module{
		id:         id,
		source:     source,
		lineOffset: lineOffset,
	}
}

// Add add source to module.
func (ms Modules) Add(m *Module) {
	ms[m.id] = m
}

// Get get module from Modules by id.
func (ms Modules) Get(id string) *Module {
	return ms[id]
}

// RequireDelegateFunc delegate func for require.
//export RequireDelegateFunc
func RequireDelegateFunc(handler unsafe.Pointer, filename *C.char, lineOffset *C.size_t) *C.char {
	id := C.GoString(filename)

	e := getEngineByEngineHandler(handler)
	if e == nil {
		logger.WithFields(logger.Fields{
			"filename": id,
		}).Error("require delegate handler does not found.")
		return nil
	}

	module := e.modules.Get(id)
	if module == nil {
		return nil
	}

	*lineOffset = C.size_t(module.lineOffset)
	cSource := C.CString(module.source)
	return cSource
}

// AttachLibVersionDelegateFunc delegate func for lib version choose
//export AttachLibVersionDelegateFunc
func AttachLibVersionDelegateFunc(handler unsafe.Pointer, require *C.char) *C.char {
	libname := C.GoString(require)
	e := getEngineByEngineHandler(handler)
	if e == nil {
		logger.WithFields(logger.Fields{
			"libname": libname,
		}).Error("delegate handler does not found.")
		return nil
	}
	if len(libname) == 0 {
		logger.Error("libname is empty.")
		return nil
	}
	return attachDefaultVersionLib(libname)
}

func attachDefaultVersionLib(libname string) *C.char {
	// block created before core.V8JSLibVersionControlHeight, default lib version: 1.0.0
	if strings.HasPrefix(libname, JSLibRootName) {
		return C.CString(libname)
	}

	return C.CString(JSLibRootName + libname)
}

func reformatModuleID(id string) string {
	paths := make([]string, 0)
	for _, p := range strings.Split(id, "/") {
		if len(p) == 0 || strings.Compare(".", p) == 0 {
			continue
		}
		if strings.Compare("..", p) == 0 {
			if len(paths) > 0 {
				paths = paths[:len(paths)-1]
				continue
			}
		}
		paths = append(paths, p)
	}

	return strings.Join(paths, "/")
}
