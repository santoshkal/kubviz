package main

import (
	"encoding/json"
	"github.com/aquasecurity/trivy/pkg/k8s/report"
	"github.com/google/uuid"
	"github.com/intelops/kubviz/constants"
	"github.com/intelops/kubviz/model"
	"github.com/nats-io/nats.go"
	"log"
	"strings"
	"sync"
)

func RunTrivyK8sClusterScan(wg *sync.WaitGroup, js nats.JetStreamContext, errCh chan error) {
	defer wg.Done()
	var report report.ConsolidatedReport
	out, err := executeCommand("trivy k8s --report summary cluster --timeout 60m -f json -q --cache-dir /tmp/.cache")
	parts := strings.SplitN(out, "{", 2)
	if len(parts) <= 1 {
		log.Println("No output from command", err)
		errCh <- err
		return
	}
	log.Println("Command logs", parts[0])
	jsonPart := "{" + parts[1]
	log.Println("First 200 lines output", jsonPart[:200])
	log.Println("Last 200 lines output", jsonPart[len(jsonPart)-200:])
	err = json.Unmarshal([]byte(jsonPart), &report)
	if err != nil {
		log.Printf("Error occurred while Unmarshalling json: %v", err)
		errCh <- err
	}
	publishTrivyK8sReport(report, js, errCh)
}

func publishTrivyK8sReport(report report.ConsolidatedReport, js nats.JetStreamContext, errCh chan error) {
	metrics := model.Trivy{
		ID:          uuid.New().String(),
		ClusterName: ClusterName,
		Report:      report,
	}
	metricsJson, _ := json.Marshal(metrics)
	_, err := js.Publish(constants.TRIVY_K8S_SUBJECT, metricsJson)
	if err != nil {
		errCh <- err
	}
	log.Printf("Trivy report with ID:%s has been published\n", metrics.ID)
	errCh <- nil
}
