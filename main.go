/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/cloudevents/sdk-go/observability/opencensus/v2/client"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	"go.uber.org/zap"
	"knative.dev/pkg/tracing"
	"knative.dev/pkg/tracing/config"
)

const (
	// HTTP path of the health endpoint used for probing the service.
	healthzPath = "/healthz"
	// Environment variable contining the tracing configuration.
	cfgTracingEnv = "K_CONFIG_TRACING"
)

func main() {
	ctx := context.Background()
	handler := NewStatefulHandler(ctx)
	run(ctx, handler)
}

func run(ctx context.Context, handler *StatefulHandler) {
	c, err := client.NewClientHTTP(
		[]cehttp.Option{cehttp.WithMiddleware(healthzMiddleware)}, nil,
	)
	if err != nil {
		log.Fatal("Failed to create client: ", err)
	}
	conf, err := config.JSONToTracingConfig(os.Getenv(cfgTracingEnv))
	if err != nil {
		log.Printf("Failed to read tracing config, using the no-op default: %v", err)
	}
	if err := tracing.SetupStaticPublishing(zap.L().Sugar(), "", conf); err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}

	if err := c.StartReceiver(ctx, handler.Handle); err != nil {
		log.Fatal("Error during receiver's runtime: ", err)
	}
}

// healthzMiddleware is a cehttp.Middleware which exposes a health endpoint.
func healthzMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.RequestURI == healthzPath {
			w.WriteHeader(http.StatusNoContent)
		} else {
			next.ServeHTTP(w, req)
		}
	})
}
