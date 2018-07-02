// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package JsonRpc

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

var (
	// Precompute the reflect.Type of error and http.Request
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfRequest = reflect.TypeOf((*http.Request)(nil)).Elem()
)

// ----------------------------------------------------------------------------
// service
// ----------------------------------------------------------------------------

type service struct {
	name     string                    // name of service
	rcvr     reflect.Value             // receiver of methods for the service
	rcvrType reflect.Type              // type of the receiver
	methods  map[string]*serviceMethod // registered methods
}

type serviceMethod struct {
	method    reflect.Method // receiver method
	argsType  reflect.Type   // type of the request argument
	replyType reflect.Type   // type of the response argument
}

// ----------------------------------------------------------------------------
// serviceMap
// ----------------------------------------------------------------------------

// serviceMap is a registry for services.
type serviceMap struct {
	mutex    sync.Mutex
	services map[string]*service
}

// register adds a new service using reflection to extract its methods.
func (m *serviceMap) register(rcvr interface{}, name string) error {
	// Setup service.
	s := &service{
		name:     name,
		rcvr:     reflect.ValueOf(rcvr),
		rcvrType: reflect.TypeOf(rcvr),
		methods:  make(map[string]*serviceMethod),
	}
	if name == "" {
		s.name = reflect.Indirect(s.rcvr).Type().Name()
		if !isExported(s.name) {
			return fmt.Errorf("rpc: type %q is not exported", s.name)
		}
	}
	if s.name == "" {
		return fmt.Errorf("rpc: no service name for type %q",
			s.rcvrType.String())
	}
	// Setup methods.
	for i := 0; i < s.rcvrType.NumMethod(); i++ {
		method := s.rcvrType.Method(i)
		mtype := method.Type

		log.Printf("[ServiceMap.register] Testing method %s", method.Name)
		// Method must be exported.
		if method.PkgPath != "" {
			log.Printf("[ServiceMap.register] %s not exported", method.Name)
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			log.Printf("[ServiceMap.register] %s needs 3 args, habe %d", method.Name, mtype.NumIn())
			continue
		}

		// First argument must be a pointer and must be exported.
		args := mtype.In(1)
		if args.Kind() != reflect.Ptr || !isExportedOrBuiltin(args) {
			log.Printf("[ServiceMap.register] %s first arg not a pointer", method.Name)
			continue
		}
		// Second argument must be a pointer and must be exported.
		reply := mtype.In(2)
		if reply.Kind() != reflect.Ptr || !isExportedOrBuiltin(args) {
			log.Printf("[ServiceMap.register] %s second arg not a pointer", method.Name)
			continue
		}
		// Method needs one out: error.
		if mtype.NumOut() != 1 {
			log.Printf("[ServiceMap.register] %s needs an out", method.Name)
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfError {
			log.Printf("[ServiceMap.register] %s needs an error out", method.Name)
			continue
		}
		s.methods[method.Name] = &serviceMethod{
			method:    method,
			argsType:  args.Elem(),
			replyType: reply.Elem(),
		}
		log.Printf("[ServiceMap.register] Found method %s", method.Name)
	}
	if len(s.methods) == 0 {
		return fmt.Errorf("rpc: %q has no exported methods of suitable type",
			s.name)
	}
	// Add to the map.
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.services == nil {
		m.services = make(map[string]*service)
	} else if _, ok := m.services[s.name]; ok {
		return fmt.Errorf("rpc: service already defined: %q", s.name)
	}
	log.Printf("[ServiceMap.register] Adding service %s", s.name)
	m.services[s.name] = s
	return nil
}

// get returns a registered service given a method name.
//
// The method name uses a dotted notation as in "Service.Method".
func (m *serviceMap) get(method string) (*service, *serviceMethod, error) {
	parts := strings.Split(method, ".")
	var method_ns = "Service1"
	var method_name = parts[0]
	if len(parts) == 2 {
		method_ns = parts[0]
		method_name = parts[1]
	}
	m.mutex.Lock()
	service := m.services[method_ns]
	m.mutex.Unlock()
	if service == nil {
		err := fmt.Errorf("rpc: can't find service %q", method_ns)
		return nil, nil, err
	}
	serviceMethod := service.methods[method_name]
	if serviceMethod == nil {
		err := fmt.Errorf("rpc: can't find method %q", method)
		return nil, nil, err
	}
	return service, serviceMethod, nil
}

// isExported returns true of a string is an exported (upper case) name.
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin.
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}