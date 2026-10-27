// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package transport

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// The service both servers serve: geth's by reflection, the module's by
// registration.

type referenceService struct {
	slow time.Duration
}

func (service *referenceService) Echo(number uint64, label *string) []any {
	return []any{number, label}
}

func (service *referenceService) Fail() error {
	return codedError{}
}

func (service *referenceService) Plain() error {
	return errors.New("plain failure")
}

func (service *referenceService) Nothing() {}

func (service *referenceService) Crash() error {
	panic("crash")
}

func (service *referenceService) Big() string {
	return strings.Repeat("x", 600)
}

func (service *referenceService) Slow(ctx context.Context) string {
	// Like an EVM run on an interpreter that does not poll, it ignores the
	// cancellation of its context.
	time.Sleep(service.slow)
	return "late"
}

type codedError struct{}

func (codedError) Error() string  { return "coded failure" }
func (codedError) ErrorCode() int { return 3 }
func (codedError) ErrorData() any { return "0xdead" }

func registerModuleService(dispatcher *dispatch.Dispatcher, slow time.Duration) {
	dispatcher.Add("test_echo", func(_ context.Context, arguments dispatch.Arguments) (any, error) {
		return []any{dispatch.Argument[uint64](arguments, 0), dispatch.Argument[*string](arguments, 1)}, nil
	}, dispatch.Required[uint64]("number"), dispatch.Optional[string]("label"))
	dispatcher.Add("test_fail", func(context.Context, dispatch.Arguments) (any, error) { return nil, codedError{} })
	dispatcher.Add("test_plain", func(context.Context, dispatch.Arguments) (any, error) { return nil, errors.New("plain failure") })
	dispatcher.Add("test_nothing", func(context.Context, dispatch.Arguments) (any, error) { return nil, nil })
	dispatcher.Add("test_crash", func(context.Context, dispatch.Arguments) (any, error) { panic("crash") })
	dispatcher.Add("test_big", func(context.Context, dispatch.Arguments) (any, error) { return strings.Repeat("x", 600), nil })
	dispatcher.Add("test_slow", func(context.Context, dispatch.Arguments) (any, error) {
		time.Sleep(slow)
		return "late", nil
	})
}

// endpoints are geth's server and the module's endpoint over the same
// service, both behind an HTTP server with the same limits.
type endpoints struct {
	reference *httptest.Server
	module    *httptest.Server
}

type endpointOptions struct {
	writeTimeout time.Duration
	slow         time.Duration
	batch        BatchLimits
	virtualHosts []string
	cors         []string
}

func newEndpoints(t *testing.T, options endpointOptions) endpoints {
	t.Helper()
	if options.virtualHosts == nil {
		options.virtualHosts = []string{"localhost"}
	}
	gethServer := gethrpc.NewServer()
	gethServer.SetBatchLimits(options.batch.RequestLimit, options.batch.ResponseMaxSize)
	require.NoError(t, gethServer.RegisterName("test", &referenceService{slow: options.slow}))
	reference := httptest.NewUnstartedServer(node.NewHTTPHandlerStack(gethServer, options.cors, options.virtualHosts, nil))
	reference.Config.WriteTimeout = options.writeTimeout
	reference.Start()
	t.Cleanup(reference.Close)
	t.Cleanup(gethServer.Stop)

	dispatcher := dispatch.New(dispatch.Config{RequestTimeout: time.Minute}, dispatch.NewMetrics(prometheus.NewRegistry()))
	t.Cleanup(dispatcher.Stop)
	registerModuleService(dispatcher, options.slow)
	front := NewFront(dispatcher, nil)
	config := HTTPConfig{VirtualHosts: options.virtualHosts, CORSOrigins: options.cors, Batch: options.batch}
	module := httptest.NewUnstartedServer(newHTTPHandler(front, config, options.writeTimeout))
	module.Config.WriteTimeout = options.writeTimeout
	module.Start()
	t.Cleanup(module.Close)
	return endpoints{reference: reference, module: module}
}

type exchange struct {
	status   int
	headers  http.Header
	body     string
	transfer []string
}

// noReuse sends each request on a connection of its own, so that a
// connection one case leaves closed cannot fail the next.
// A request with "Expect: 100-continue" waits for the server before it sends
// its body, so that a refusal arrives before the body.
var noReuse = &http.Transport{DisableKeepAlives: true, DisableCompression: true, ExpectContinueTimeout: 5 * time.Second}

func post(t *testing.T, server *httptest.Server, request func() *http.Request) exchange {
	t.Helper()
	response, err := noReuse.RoundTrip(request())
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	if response.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		body, err = io.ReadAll(reader)
		require.NoError(t, err)
	}
	return exchange{status: response.StatusCode, headers: response.Header, body: string(body), transfer: response.TransferEncoding}
}

func jsonRequest(server *httptest.Server, body string) func() *http.Request {
	return func() *http.Request {
		request, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		return request
	}
}

// parameterWording matches the wording of a parameter the module could not
// decode, which differs from go-ethereum's (plan, section 6).
var parameterWording = regexp.MustCompile(`"message":"(invalid argument \d+|parameter \d+ \(\w+\))[^"]*"`)

func normalize(body string) string {
	return parameterWording.ReplaceAllString(body, `"message":"undecodable parameter"`)
}

// requireSameExchange sends the same request to both servers and compares
// status, content type, encoding and body byte for byte.
func requireSameExchange(t *testing.T, servers endpoints, name string, build func(server *httptest.Server) func() *http.Request) {
	t.Helper()
	reference := post(t, servers.reference, build(servers.reference))
	module := post(t, servers.module, build(servers.module))
	require.Equal(t, reference.status, module.status, "%s: status", name)
	require.Equal(t, reference.headers.Get("Content-Type"), module.headers.Get("Content-Type"), "%s: content type", name)
	require.Equal(t, reference.headers.Get("Content-Encoding"), module.headers.Get("Content-Encoding"), "%s: content encoding", name)
	require.Equal(t, normalize(reference.body), normalize(module.body), "%s: body", name)
}

// bodies are the requests the envelope test sends, each a case of the
// JSON-RPC 2.0 envelope go-ethereum's server answers in a particular way.
var bodies = map[string]string{
	"call":                         `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[5,"five"]}`,
	"call without optional":        `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[5]}`,
	"call with null optional":      `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[5,null]}`,
	"string id":                    `{"jsonrpc":"2.0","id":"abc","method":"test_echo","params":[1]}`,
	"null id":                      `{"jsonrpc":"2.0","id":null,"method":"test_echo","params":[1]}`,
	"object id":                    `{"jsonrpc":"2.0","id":{},"method":"test_echo","params":[1]}`,
	"array id":                     `{"jsonrpc":"2.0","id":[1],"method":"test_echo","params":[1]}`,
	"float id":                     `{"jsonrpc":"2.0","id":1.5,"method":"test_echo","params":[1]}`,
	"no version":                   `{"id":1,"method":"test_echo","params":[1]}`,
	"wrong version":                `{"jsonrpc":"1.0","id":1,"method":"test_echo","params":[1]}`,
	"no method":                    `{"jsonrpc":"2.0","id":1,"params":[1]}`,
	"unknown method":               `{"jsonrpc":"2.0","id":1,"method":"test_unknown","params":[]}`,
	"method without namespace":     `{"jsonrpc":"2.0","id":1,"method":"echo","params":[]}`,
	"unknown namespace":            `{"jsonrpc":"2.0","id":1,"method":"other_echo","params":[]}`,
	"method name too long":         `{"jsonrpc":"2.0","id":1,"method":"test_` + strings.Repeat("e", 2100) + `","params":[]}`,
	"notification":                 `{"jsonrpc":"2.0","method":"test_echo","params":[1]}`,
	"response":                     `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
	"error response":               `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"x"}}`,
	"params object":                `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":{"number":1}}`,
	"params string":                `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":"1"}`,
	"params null":                  `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":null}`,
	"params absent":                `{"jsonrpc":"2.0","id":1,"method":"test_echo"}`,
	"too many params":              `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1,"x",2]}`,
	"undecodable param":            `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x"]}`,
	"coded error with data":        `{"jsonrpc":"2.0","id":1,"method":"test_fail","params":[]}`,
	"plain error":                  `{"jsonrpc":"2.0","id":1,"method":"test_plain","params":[]}`,
	"null result":                  `{"jsonrpc":"2.0","id":1,"method":"test_nothing","params":[]}`,
	"panic":                        `{"jsonrpc":"2.0","id":1,"method":"test_crash","params":[]}`,
	"html in result":               `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1,"<b>&</b>"]}`,
	"not json":                     `{"jsonrpc":`,
	"empty body":                   ``,
	"whitespace body":              `   `,
	"json null":                    `null`,
	"json number":                  `5`,
	"json string":                  `"x"`,
	"trailing data":                `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]} garbage`,
	"batch":                        `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]},{"jsonrpc":"2.0","id":2,"method":"test_fail"}]`,
	"batch with notification":      `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]},{"jsonrpc":"2.0","method":"test_echo","params":[2]}]`,
	"batch of notifications":       `[{"jsonrpc":"2.0","method":"test_echo","params":[1]},{"jsonrpc":"2.0","method":"test_echo","params":[2]}]`,
	"batch with invalid elements":  `[1, null, "x", {}, {"jsonrpc":"2.0","id":3,"method":"test_echo","params":[3]}]`,
	"batch with responses":         `[{"jsonrpc":"2.0","id":1,"result":1},{"jsonrpc":"2.0","id":2,"method":"test_echo","params":[2]}]`,
	"batch of responses":           `[{"jsonrpc":"2.0","id":1,"result":1}]`,
	"empty batch":                  `[]`,
	"nested batch":                 `[[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]}]]`,
	"batch with unknown method":    `[{"jsonrpc":"2.0","id":1,"method":"test_unknown"}]`,
	"batch with panic":             `[{"jsonrpc":"2.0","id":1,"method":"test_crash"},{"jsonrpc":"2.0","id":2,"method":"test_echo","params":[2]}]`,
	"subscription over http":       `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`,
	"other subscription over http": `{"jsonrpc":"2.0","id":1,"method":"test_subscribe","params":["anything"]}`,
	"unsubscription over http":     `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":["0x1"]}`,
	"unsubscription without id":    `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":[]}`,
	"unsubscription of a number":   `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":[1]}`,
	"long unsubscription name":     `{"jsonrpc":"2.0","id":1,"method":"` + strings.Repeat("e", 2100) + `_unsubscribe","params":["0x1"]}`,
}

func TestEnvelope_MatchesGoEthereum(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{})
	for name, body := range bodies {
		requireSameExchange(t, servers, name, func(server *httptest.Server) func() *http.Request {
			return jsonRequest(server, body)
		})
	}
}

func TestEnvelope_MatchesGoEthereumWithGzip(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{})
	for _, name := range []string{"call", "batch", "not json", "empty batch"} {
		requireSameExchange(t, servers, name, func(server *httptest.Server) func() *http.Request {
			return func() *http.Request {
				request := jsonRequest(server, bodies[name])()
				request.Header.Set("Accept-Encoding", "gzip")
				return request
			}
		})
	}
}

func TestEnvelope_MatchesGoEthereumUnderBatchLimits(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{batch: BatchLimits{RequestLimit: 3, ResponseMaxSize: 1000}})
	calls := func(count int, method string) string {
		var elements []string
		for index := range count {
			elements = append(elements, `{"jsonrpc":"2.0","id":`+strings.Repeat("1", index+1)+`,"method":"`+method+`","params":[]}`)
		}
		return "[" + strings.Join(elements, ",") + "]"
	}
	cases := map[string]string{
		"at the item limit":            calls(3, "test_nothing"),
		"over the item limit":          calls(4, "test_nothing"),
		"over the item limit, no call": `[1,2,3,4]`,
		"over the response size":       calls(3, "test_big"),
		"under the response size":      calls(1, "test_big"),
	}
	for name, body := range cases {
		requireSameExchange(t, servers, name, func(server *httptest.Server) func() *http.Request {
			return jsonRequest(server, body)
		})
	}
}

// TestEnvelope_MatchesGoEthereumAtTheWriteTimeout holds D3: a call still
// running at the write timeout minus 100ms is answered -32002, and so is
// every unanswered call of a batch.
func TestEnvelope_MatchesGoEthereumAtTheWriteTimeout(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{writeTimeout: 1100 * time.Millisecond, slow: 1500 * time.Millisecond})
	cases := map[string]string{
		"a slow call": `{"jsonrpc":"2.0","id":1,"method":"test_slow","params":[]}`,
		"a batch with fast and slow calls": `[{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]},` +
			`{"jsonrpc":"2.0","id":2,"method":"test_slow"},{"jsonrpc":"2.0","id":3,"method":"test_echo","params":[3]},` +
			`{"jsonrpc":"2.0","method":"test_echo","params":[4]}]`,
		"a fast call": `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":[1]}`,
	}
	for name, body := range cases {
		requireSameExchange(t, servers, name, func(server *httptest.Server) func() *http.Request {
			return jsonRequest(server, body)
		})
	}
}

// TestEnvelope_ASlowNotificationIsAnswered: go-ethereum answers a
// notification still running at the write timeout too, though it races the
// end of the request doing so; the module writes the answer exactly once.
func TestEnvelope_ASlowNotificationIsAnswered(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{writeTimeout: 1100 * time.Millisecond, slow: 1500 * time.Millisecond})
	module := post(t, servers.module, jsonRequest(servers.module, `{"jsonrpc":"2.0","method":"test_slow","params":[]}`))
	require.Equal(t, http.StatusOK, module.status)
	require.Equal(t, `{"jsonrpc":"2.0","error":{"code":-32002,"message":"request timed out"}}`, module.body)
	require.Equal(t, "", module.headers.Get("Content-Encoding"))
}

func TestHTTP_MatchesGoEthereumBeforeTheEnvelope(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{cors: []string{"https://allowed.example"}})
	request := func(method, contentType, body string, mutate ...func(*http.Request)) func(*httptest.Server) func() *http.Request {
		return func(server *httptest.Server) func() *http.Request {
			return func() *http.Request {
				built, _ := http.NewRequest(method, server.URL, strings.NewReader(body))
				if contentType != "" {
					built.Header.Set("Content-Type", contentType)
				}
				for _, change := range mutate {
					change(built)
				}
				return built
			}
		}
	}
	cases := map[string]func(*httptest.Server) func() *http.Request{
		"health check of a load balancer": request(http.MethodGet, "", ""),
		"get with a query":                request(http.MethodGet, "", "", func(built *http.Request) { built.URL.RawQuery = "x=1" }),
		"put":                             request(http.MethodPut, "application/json", bodies["call"]),
		"delete":                          request(http.MethodDelete, "application/json", bodies["call"]),
		"no content type":                 request(http.MethodPost, "", bodies["call"]),
		"text content type":               request(http.MethodPost, "text/plain", bodies["call"]),
		"json-rpc content type":           request(http.MethodPost, "application/json-rpc", bodies["call"]),
		"jsonrequest content type":        request(http.MethodPost, "application/jsonrequest", bodies["call"]),
		"content type with charset":       request(http.MethodPost, "application/json; charset=utf-8", bodies["call"]),
		"options":                         request(http.MethodOptions, "", ""),
		"body over the limit": request(http.MethodPost, "application/json", "", func(built *http.Request) {
			built.Header.Set("Expect", "100-continue")
			built.ContentLength = httpBodyLimit + 1
			built.Body = io.NopCloser(strings.NewReader(strings.Repeat(" ", httpBodyLimit+1)))
		}),
		"chunked body over the limit": request(http.MethodPost, "application/json", "", func(built *http.Request) {
			built.ContentLength = -1
			built.Body = io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["` + strings.Repeat("x", httpBodyLimit) + `"]}`))
		}),
		"allowed host":       request(http.MethodPost, "application/json", bodies["call"], func(built *http.Request) { built.Host = "localhost:8545" }),
		"disallowed host":    request(http.MethodPost, "application/json", bodies["call"], func(built *http.Request) { built.Host = "attacker.example" }),
		"host as an address": request(http.MethodPost, "application/json", bodies["call"], func(built *http.Request) { built.Host = "10.0.0.1" }),
		"allowed origin": request(http.MethodPost, "application/json", bodies["call"], func(built *http.Request) {
			built.Header.Set("Origin", "https://allowed.example")
		}),
		"disallowed origin": request(http.MethodPost, "application/json", bodies["call"], func(built *http.Request) {
			built.Header.Set("Origin", "https://attacker.example")
		}),
		"preflight": request(http.MethodOptions, "", "", func(built *http.Request) {
			built.Header.Set("Origin", "https://allowed.example")
			built.Header.Set("Access-Control-Request-Method", "POST")
		}),
	}
	for name, build := range cases {
		reference := post(t, servers.reference, build(servers.reference))
		module := post(t, servers.module, build(servers.module))
		require.Equal(t, reference.status, module.status, "%s: status", name)
		require.Equal(t, normalize(reference.body), normalize(module.body), "%s: body", name)
		for _, header := range []string{"Content-Type", "Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Vary"} {
			require.Equal(t, reference.headers.Get(header), module.headers.Get(header), "%s: header %s", name, header)
		}
	}
}

// TestEnvelope_TheModuleServesGethsClient: go-ethereum's client, the one the
// node's own tools use, works against the module's endpoint.
func TestEnvelope_TheModuleServesGethsClient(t *testing.T) {
	servers := newEndpoints(t, endpointOptions{})
	client, err := gethrpc.Dial(servers.module.URL)
	require.NoError(t, err)
	defer client.Close()
	var echoed []any
	require.NoError(t, client.Call(&echoed, "test_echo", 7, "seven"))
	require.Equal(t, []any{float64(7), "seven"}, echoed)

	err = client.Call(nil, "test_fail")
	var coded gethrpc.DataError
	require.ErrorAs(t, err, &coded)
	require.Equal(t, "0xdead", coded.ErrorData())

	batch := []gethrpc.BatchElem{
		{Method: "test_echo", Args: []any{1}, Result: new([]any)},
		{Method: "test_unknown", Result: new(any)},
	}
	require.NoError(t, client.BatchCall(batch))
	require.NoError(t, batch[0].Error)
	require.Error(t, batch[1].Error)

	_, err = client.Subscribe(context.Background(), "eth", make(chan any), "newHeads")
	require.ErrorIs(t, err, gethrpc.ErrNotificationsUnsupported)
}

func TestRPCModules_ListsWhatTheTransportServes(t *testing.T) {
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(prometheus.NewRegistry()))
	t.Cleanup(dispatcher.Stop)
	registerModuleService(dispatcher, 0)
	dispatcher.Add("eth_chainId", func(context.Context, dispatch.Arguments) (any, error) { return "0x1", nil })
	front := NewFront(dispatcher, map[string][]string{HTTP: {"test"}})

	modules, err := front.Route(context.Background(), HTTP, "rpc_modules", nil)
	require.Nil(t, err)
	require.JSONEq(t, `{"rpc":"1.0","test":"1.0"}`, string(modules))
	modules, err = front.Route(context.Background(), IPC, "rpc_modules", nil)
	require.Nil(t, err)
	require.JSONEq(t, `{"rpc":"1.0","test":"1.0","eth":"1.0"}`, string(modules))

	_, err = front.Route(context.Background(), HTTP, "eth_chainId", nil)
	require.Equal(t, dispatch.CodeMethodNotFound, err.Code, "a namespace not enabled on the transport is not served")
}

type recordingForwarder struct {
	calls []string
}

func (forwarder *recordingForwarder) Forward(_ context.Context, method string, parameters []json.RawMessage) (json.RawMessage, error) {
	if method == "rpc_modules" {
		return json.RawMessage(`{"dag":"1.0","eth":"1.0","admin":"1.0"}`), nil
	}
	encoded, _ := json.Marshal(parameters)
	forwarder.calls = append(forwarder.calls, method+string(encoded))
	return json.RawMessage(`"forwarded"`), nil
}

// TestFront_ForwardsOnIPCAndThePeerFactsEverywhere holds S1 and X3: the
// node's handler serves the legacy namespaces on IPC only, and the p2p facts
// on every transport.
func TestFront_ForwardsOnIPCAndThePeerFactsEverywhere(t *testing.T) {
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(prometheus.NewRegistry()))
	t.Cleanup(dispatcher.Stop)
	registerModuleService(dispatcher, 0)
	forwarder := &recordingForwarder{}
	front := NewFront(dispatcher, map[string][]string{HTTP: {"test", "dag", "eth", "net"}, WebSocket: {"dag", "net"}})
	front.Forward(forwarder)

	result, err := front.Route(context.Background(), IPC, "dag_getHeads", json.RawMessage(`["latest"]`))
	require.Nil(t, err)
	require.Equal(t, `"forwarded"`, string(result))
	for _, transport := range []string{HTTP, WebSocket} {
		_, err = front.Route(context.Background(), transport, "dag_getHeads", json.RawMessage(`["latest"]`))
		require.Equal(t, dispatch.MethodNotFound("dag_getHeads"), err, "a legacy method on %s", transport)
		result, err = front.Route(context.Background(), transport, "net_peerCount", nil)
		require.Nil(t, err)
		require.Equal(t, `"forwarded"`, string(result), "a p2p fact on %s", transport)
	}
	result, err = front.Route(context.Background(), IPC, "test_echo", json.RawMessage(`[1]`))
	require.Nil(t, err)
	require.Equal(t, `[1,null]`, string(result), "a method the module serves is not forwarded")
	_, err = front.Route(context.Background(), HTTP, "net_listening", nil)
	require.Nil(t, err)
	_, err = front.Route(context.Background(), HTTP, "admin_peers", nil)
	require.Equal(t, dispatch.CodeMethodNotFound, err.Code, "a namespace not enabled is not forwarded")
	_, err = front.Route(context.Background(), IPC, "dag_getHeads", json.RawMessage(`{"epoch":1}`))
	require.Equal(t, &dispatch.Error{Code: dispatch.CodeInvalidParameters, Message: "non-array args"}, err)
	require.Equal(t, []string{`dag_getHeads["latest"]`, `net_peerCountnull`, `net_peerCountnull`, `net_listeningnull`}, forwarder.calls)

	modules, err := front.Route(context.Background(), HTTP, "rpc_modules", nil)
	require.Nil(t, err)
	require.JSONEq(t, `{"rpc":"1.0","test":"1.0"}`, string(modules), "the node's namespaces are not listed off IPC")
	modules, err = front.Route(context.Background(), IPC, "rpc_modules", nil)
	require.Nil(t, err)
	require.JSONEq(t, `{"rpc":"1.0","test":"1.0","dag":"1.0","eth":"1.0","admin":"1.0"}`, string(modules))
}

// TestFront_AMethodNoRegisterNamesIsNotFoundOnEveryTransport holds S1.
func TestFront_AMethodNoRegisterNamesIsNotFoundOnEveryTransport(t *testing.T) {
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(prometheus.NewRegistry()))
	t.Cleanup(dispatcher.Stop)
	registerModuleService(dispatcher, 0)
	front := NewFront(dispatcher, nil)
	for _, transport := range []string{HTTP, WebSocket, IPC} {
		for _, method := range []string{"test_Echo", "test_register", "test_ServeHTTP", "test_", "_echo", "test"} {
			_, err := front.Route(context.Background(), transport, method, nil)
			require.Equal(t, dispatch.MethodNotFound(method), err, "%s on %s", method, transport)
		}
	}
}
