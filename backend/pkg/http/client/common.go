package http_client

import (
	"context"

	"github.com/go-resty/resty/v2"
	internalhttp "github.com/olga-larina/otus-highload/pkg/http"
	"github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/pkg/tracing"
	"go.opentelemetry.io/otel"
)

type HttpClient struct {
	delegate *resty.Client
}

func NewHttpClient(delegate *resty.Client) *HttpClient {
	return &HttpClient{delegate: delegate}
}

func (t *HttpClient) Get(ctx context.Context, path string, pathParams map[string]string, request any, responseTemplate any) (*resty.Response, error) {
	return t.execute(ctx, resty.MethodGet, path, pathParams, request, responseTemplate)
}

func (t *HttpClient) Post(ctx context.Context, path string, pathParams map[string]string, request any, responseTemplate any) (*resty.Response, error) {
	return t.execute(ctx, resty.MethodPost, path, pathParams, request, responseTemplate)
}

func (t *HttpClient) execute(ctx context.Context, method string, path string, pathParams map[string]string, request any, responseTemplate any) (*resty.Response, error) {
	ctxWithSpan, span := otel.Tracer("default").Start(ctx, path)
	defer span.End()

	token, ok := ctxWithSpan.Value(model.TokenContextKey).(string)
	if !ok {
		token = ""
	}
	return t.delegate.R().
		// SetDebug(true).
		SetContext(ctxWithSpan).
		SetHeader(internalhttp.HEADER_REQUEST_ID, tracing.GetRequestId(ctxWithSpan)).
		SetHeader(internalhttp.HEADER_TRACE_ID, tracing.GetTraceId(ctxWithSpan)).
		SetHeader(internalhttp.HEADER_SPAN_ID, tracing.GetSpanId(ctxWithSpan)).
		SetHeader(internalhttp.HEADER_AUTHORIZATION, token).
		SetPathParams(pathParams).
		SetBody(request).
		SetResult(responseTemplate).
		Execute(method, path)
}
