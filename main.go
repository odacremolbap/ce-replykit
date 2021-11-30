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
	"os"
	"strconv"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

const (
	storageTTLEnv = "CE_REPLY_KIT_STORAGE_TTL_SECONDS"
)

func main() {
	ctx := context.Background()

	c, err := cloudevents.NewClientHTTP(cehttp.WithPort(8080))
	if err != nil {
		log.Fatal("Failed to create CloudEvents client: ", err)
	}

	storageTTL := 300 * time.Second
	st := os.Getenv(storageTTLEnv)
	if st != "" {
		i, err := strconv.Atoi(st)
		if err != nil {
			log.Fatalf("Storage TTL provided via %q is not a number: %v", storageTTLEnv, err)
		}
		storageTTL = time.Duration(i) * time.Second
	}

	handler := NewRequestHandler(ctx, storageTTL)
	if err := c.StartReceiver(ctx, handler.Handle); err != nil {
		log.Fatal("Error during receiver's runtime: ", err)
	}
}
