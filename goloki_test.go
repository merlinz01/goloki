package goloki

import (
	"testing"
	"time"
)

func TestStreamMessageFormat(t *testing.T) {
	logger := LokiLogger{}
	logger.LokiUrl = "http://localhost:3100/loki/api/v1/push"
	logger.MetadataLabels = append(logger.MetadataLabels, "path")
	logger.MetadataLabels = append(logger.MetadataLabels, "statuscode")
	logger.Setup()

	message := NewMessage("192.168.1.1 GET /index.html -> 200 (295 B)")
	message.Metadata["client"] = "192.168.1.1"
	message.Metadata["method"] = "GET"
	message.Metadata["path"] = "/index.html"
	message.Metadata["statuscode"] = "200"
	message.Metadata["size"] = "295"
	message.timestamp = time.Time{}
	result := string(logger.formatMessageStream(&message))
	expected := `{ "stream": {"path":"/index.html","statuscode":"200"}, "values": [ [ "-6795364578871345152", "192.168.1.1 GET /index.html -\u003e 200 (295 B)", {"client":"192.168.1.1","method":"GET","size":"295"} ] ] } `
	if result != expected {
		println(result)
		t.Error("Invalid format result")
	}
	logger.Shutdown()
}
