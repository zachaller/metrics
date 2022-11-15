package prometheus

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"time"
)

type PrometheusProvider struct {
	client api.Client
	v1api  v1.API
}

func NewPrometheus(address string) (PrometheusProvider, error) {
	client, err := api.NewClient(api.Config{
		Address: address, // http://demo.robustperception.io:9090
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return PrometheusProvider{}, err
	}
	v1api := v1.NewAPI(client)

	return PrometheusProvider{
		client: client,
		v1api:  v1api,
	}, nil
}

// Query performs a query agains prometheus server
func (p *PrometheusProvider) Query(ctx context.Context, query string, start time.Time, end time.Time, step time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, warnings, err := p.v1api.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	fmt.Printf("Result:\n%v\n", result)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
