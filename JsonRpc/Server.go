// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package JsonRpc

import (
	"fmt"
	"log"
	"reflect"
)

// ----------------------------------------------------------------------------

type Request struct {
	Body string
}
type Response struct {
	Status int
	Body   string
}

// ----------------------------------------------------------------------------
// Codec
// ----------------------------------------------------------------------------

// Codec creates a CodecRequest to process each request.
type ICodec interface {
	NewRequest(*Request) ICodecRequest
}

// CodecRequest decodes a request and encodes a response using a specific
// serialization scheme.
type ICodecRequest interface {
	// Reads request and returns the RPC method name.
	Method() (string, error)
	// Reads request filling the RPC method args.
	ReadRequest(interface{}) error
	// Writes response using the RPC method reply. The error parameter is
	// the error returned by the method call, if any.
	WriteResponse(interface{}, error) (string, error)
}

// ----------------------------------------------------------------------------
// Server
// ----------------------------------------------------------------------------

// NewServer returns a new RPC server.
func NewServer() *Server {
	return &Server{
		codec:    NewJsonCodec(),
		services: new(serviceMap),
	}
}

// RequestInfo contains all the information we pass to before/after functions
type RequestInfo struct {
	Method string
	Error  error
	//Request    *http.Request
	StatusCode int
}

// Server serves registered RPC services using registered codecs.
type Server struct {
	codec      ICodec
	services   *serviceMap
	beforeFunc func(i *RequestInfo)
	afterFunc  func(i *RequestInfo)
}

// RegisterService adds a new TCP service to the server.
// No HTTP request struct will be passed to the service methods.
//
// The name parameter is optional: if empty it will be inferred from
// the receiver type name.
//
// Methods from the receiver will be extracted if these rules are satisfied:
//
//    - The receiver is exported (begins with an upper case letter) or local
//      (defined in the package registering the service).
//    - The method name is exported.
//    - The method has two arguments: *args, *reply.
//    - Both arguments are pointers.
//    - Both arguments are exported or local.
//    - The method has return type error.
//
// All other methods are ignored.
func (s *Server) RegisterService(receiver interface{}, name string) error {
	return s.services.register(receiver, name)
}

// HasMethod returns true if the given method is registered.
//
// The method uses a dotted notation as in "Service.Method".
func (s *Server) HasMethod(method string) bool {
	if _, _, err := s.services.get(method); err == nil {
		return true
	}
	return false
}

// RegisterBeforeFunc registers the specified function as the function
// that will be called before every request.
//
// Note: Only one function can be registered, subsequent calls to this
// method will overwrite all the previous functions.
func (s *Server) RegisterBeforeFunc(f func(i *RequestInfo)) {
	s.beforeFunc = f
}

// RegisterAfterFunc registers the specified function as the function
// that will be called after every request
//
// Note: Only one function can be registered, subsequent calls to this
// method will overwrite all the previous functions.
func (s *Server) RegisterAfterFunc(f func(i *RequestInfo)) {
	s.afterFunc = f
}

// Serve incoming request
func (s *Server) ServeRequest(r *Request, w *Response) {
	log.Println("[ServeRequest]")
	// Create a new codec request.
	codecReq := s.codec.NewRequest(r)
	// Get service method to be called.
	method, errMethod := codecReq.Method()
	if errMethod != nil {
		s.writeError(w, 400, errMethod.Error())
		return
	}
	serviceSpec, methodSpec, errGet := s.services.get(method)
	if errGet != nil {
		log.Printf("Error getting method '%s'. %s", method, errGet)
		s.writeError(w, 400, errGet.Error())
		return
	}
	// Decode the args.
	args := reflect.New(methodSpec.argsType)
	if errRead := codecReq.ReadRequest(args.Interface()); errRead != nil {
		log.Printf("Error reading request '%s'. %s", r.Body, errRead)
		s.writeError(w, 400, errRead.Error())
		return
	}

	// Call the registered Before Function
	if s.beforeFunc != nil {
		s.beforeFunc(&RequestInfo{
			//Request: r,
			Method: method,
		})
	}

	// Call the service method.
	log.Printf("[ServeRequest] Calling method:%s", method)
	reply := reflect.New(methodSpec.replyType)

	// omit the HTTP request if the service method doesn't accept it
	var errValue []reflect.Value
	errValue = methodSpec.method.Func.Call([]reflect.Value{
		serviceSpec.rcvr,
		args,
		reply,
	})

	// Cast the result to error if needed.
	var errResult error
	errInter := errValue[0].Interface()
	if errInter != nil {
		errResult = errInter.(error)
	}

	// Encode the response.
	if res, errWrite := codecReq.WriteResponse(reply.Interface(), errResult); errWrite != nil {
		s.writeError(w, 400, errWrite.Error())
	} else {
		w.Body = res
		// Call the registered After Function
		if s.afterFunc != nil {
			s.afterFunc(&RequestInfo{
				//Request:    r,
				Method:     method,
				Error:      errResult,
				StatusCode: 200,
			})
		}
	}
}

func (s *Server) writeError(w *Response, status int, msg string) {
	w.Body = msg
	w.Status = status
	if s.afterFunc != nil {
		s.afterFunc(&RequestInfo{
			Error:      fmt.Errorf(msg),
			StatusCode: status,
		})
	}
}
