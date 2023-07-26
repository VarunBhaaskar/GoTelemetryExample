package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	_ "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/VarunBhaaskar/GoTelemetryExample"
	instrumentationVersion = "v0.1.0"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	tp := otel.GetTracerProvider()
	fmt.Println((tp))
	log.Printf("%v %s %s %v", time.Now(), r.Method, r.URL.Path, time.Since(start))
}

func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("gotelemetryexample"),
		semconv.ServiceVersion("0.0.1"),
		attribute.String("service.name", "gotelemetryexample"),
		attribute.String("service.application", "AZA"),
		attribute.String("application.name", "gotelemetryexample"),
		attribute.String("application.id", "AZA"),
	)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(otelAgentAddr),
		otlptracehttp.WithURLPath(""),
	)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Go Server for GoTelemetryExample is up and running!\n")
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there! I am GoTelemetryExample! I take requests in \n1. / \n2. /books/{title}/page/{page}\n")
}

func errorEndpoint(w http.ResponseWriter, r *http.Request) {
	// w.WriteHeader(http.StatusInternalServerError)
	// w.Header().Set("Content-Type", "application/json")
	// resp := make(map[string]string)
	// resp["message"] = "Deliberate error raised"
	// jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	// }
	// w.Write(jsonResp)
	panic("Deliberatrely Panicing here!! ")
}

func booksPageGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	page := vars["page"]
	res := getPage(r.Context(), page)
	fmt.Fprintf(w, "You've requested the book: %s on page %s. The response is %s\n", title, page, res)
}

func getPage(ctx context.Context, id string) string {
	_, span := tracer.Start(ctx, "getPage", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	if id == "123" {
		return "Trace Testing"
	}
	return "unknown page"
}

func main() {

	errE := godotenv.Load(".env")
	if errE != nil {
		log.Fatalf("Error loading environment variables file")
	}

	tp, err := initTracer()

	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	logLoction := os.Getenv("GTM_LOGS")
	fmt.Println(logLoction)

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("gotelemetryexample"))
	wrappedMux := NewLogger(r)

	r.HandleFunc("/", rootHandler)

	r.HandleFunc("/hello", helloHandler)

	r.HandleFunc("/gopanic", errorEndpoint)

	r.HandleFunc("/books/{title}/page/{page}", booksPageGetHandler)

	fmt.Println("The server is up and running in :8080")
	http.ListenAndServe(":8080", wrappedMux)

}
