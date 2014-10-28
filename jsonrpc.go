package main

import "encoding/json"

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type JSONRPCRequest struct {
	string `json:"jsonrpc"`
	Id     string        `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type JSONRPCResponse struct {
	JsonRPCTag string        `json:"jsonrpc"`
	Id         string        `json:"id"`
	Result     interface{}   `json:"result"`
	Error      *JSONRPCError `json:"error"`
}

type JSONRPCFunction func(request JSONRPCRequest) chan JSONRPCResponse

type JSONRPCRouter struct {
	rpc_functions map[string]JSONRPCFunction
}

func (r *JSONRPCRouter) Init() {
	r.rpc_functions = make(map[string]JSONRPCFunction)
}

func (r *JSONRPCRouter) AddHandler(method string, handler JSONRPCFunction) {
	r.rpc_functions[method] = handler
}

func (r *JSONRPCRouter) Call(request JSONRPCRequest) chan JSONRPCResponse {
	f, ok := r.rpc_functions[request.Method]
	if !ok {
		response := make(chan JSONRPCResponse, 1)
		jerr := &JSONRPCError{-32601, "Method not found", nil}
		err_resp := JSONRPCResponse{"2.0", request.Id, nil, jerr}
		response <- err_resp
		logger.Debugf("Method not found: %v", err_resp)
		return response
	}
	return f(request)
}

func (r *JSONRPCRouter) CallRaw(request []byte) chan []byte {
	var jrequest JSONRPCRequest
	err := json.Unmarshal(request, &jrequest)
	var response chan JSONRPCResponse
	logger.Debugf("Data request: %v", string(request))
	if err != nil {
		logger.Debugf("Parse error: %v", err)
		response = make(chan JSONRPCResponse, 1)
		jerr := &JSONRPCError{-32700, "Parse error", nil}
		thing := JSONRPCResponse{"2.0", jrequest.Id, nil, jerr}
		response <- thing
	} else {
		response = r.Call(jrequest)
	}
	bytes_response := make(chan []byte, 1)
	go func() {
		to_marshal := <-response
		logger.Debugf("About to marshal and send response: %v", to_marshal)
		bytes, _ := json.Marshal(to_marshal)
		bytes_response <- bytes
	}()
	return bytes_response
}
