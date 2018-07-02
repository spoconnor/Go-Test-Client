package JsonRpc

import (
	"encoding/json"
	"errors"
)

var null = json.RawMessage([]byte("null"))

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

// serverRequest represents a JSON-RPC request received by the server.
type serverRequest struct {
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// An Array of objects to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	Id *json.RawMessage `json:"id"`
}

// serverResponse represents a JSON-RPC response returned by the server.
type serverResponse struct {
	// The Object that was returned by the invoked method. This must be null
	// in case there was an error invoking the method.
	Result interface{} `json:"result"`
	// An Error object if there was an error invoking the method. It must be
	// null if there was no error.
	Error interface{} `json:"error"`
	// This must be the same id as the request it is responding to.
	Id *json.RawMessage `json:"id"`
}

// ----------------------------------------------------------------------------
// JsonCodec
// ----------------------------------------------------------------------------

// NewJsonCodec returns a new JSON Codec.
func NewJsonCodec() *JsonCodec {
	return &JsonCodec{}
}

// JsonCodec creates a JsonCodecRequest to process each request.
type JsonCodec struct {
}

// NewRequest returns a JsonCodecRequest.
func (c *JsonCodec) NewRequest(request *Request) ICodecRequest {
	return newJsonCodecRequest(request)
}

// ----------------------------------------------------------------------------
// JsonCodecRequest
// ----------------------------------------------------------------------------

// newJsonCodecRequest returns a new JsonCodecRequest.
func newJsonCodecRequest(request *Request) ICodecRequest {
	// Decode the request body and check if RPC method is valid.
	req := new(serverRequest)
	err := json.Unmarshal([]byte(request.Body), req)
	return &JsonCodecRequest{request: req, err: err}
}

// CodecRequest decodes and encodes a single request.
type JsonCodecRequest struct {
	request *serverRequest
	err     error
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *JsonCodecRequest) Method() (string, error) {
	if c.err == nil {
		return c.request.Method, nil
	}
	return "", c.err
}

// ReadRequest fills the request object for the RPC method.
func (c *JsonCodecRequest) ReadRequest(args interface{}) error {
	if c.err == nil {
		if c.request.Params != nil {
			// JSON params is array value. RPC params is struct.
			// Unmarshal into array containing the request struct.
			params := [1]interface{}{args} // TODO - why an array here?
			c.err = json.Unmarshal(*c.request.Params, &params[0])
		} else {
			c.err = errors.New("rpc: method request ill-formed: missing params field")
		}
	}
	return c.err
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
//
// The err parameter is the error resulted from calling the RPC method,
// or nil if there was no error.
func (c *JsonCodecRequest) WriteResponse(reply interface{}, methodErr error) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	res := &serverResponse{
		Result: reply,
		Error:  &null,
		Id:     c.request.Id,
	}
	if methodErr != nil {
		// Propagate error message as string.
		res.Error = methodErr.Error()
		// Result must be null if there was an error invoking the method.
		// http://json-rpc.org/wiki/specification#a1.2Response
		res.Result = &null
	}
	if c.request.Id == nil {
		// Id is null for notifications and they don't have a response.
		res.Id = &null
	} else {
		data, err := json.Marshal(res)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", nil
}
