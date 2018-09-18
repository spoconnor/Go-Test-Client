package main

import (
	"errors"
	"fmt"
	"log"
	"github.com/spoconnor/Go-Test-Client/JsonRpc"
)

var ErrResponseError = errors.New("response error")

type PingRequest struct {
}
type PingResponse struct {
}

type Service1Request struct {
	A int
	B int
}

type Service1BadRequest struct {
	M string `json:"method"`
}

type Service1Response struct {
	Result int
}

type Service1 struct {
	beforeAfterContext map[string]string
}

func (t *Service1) Ping(req *PingRequest, res *PingResponse) error {
	log.Printf("Ping")
	return nil
}

func (t *Service1) Multiply(req *Service1Request, res *Service1Response) error {
	res.Result = req.A * req.B
	log.Printf("[Service1.Multiply] %d x %d = %d", req.A, req.B, res.Result)
	return nil
}

func (t *Service1) ResponseError(req *Service1Request, res *Service1Response) error {
	return ErrResponseError
}

func (t *Service1) BeforeAfter(r *JsonRpc.Request, req *Service1Request, res *Service1Response) error {
	if _, ok := t.beforeAfterContext["before"]; !ok {
		return fmt.Errorf("before value not found in context")
	}
	res.Result = 1
	return nil
}
