# GoLoki

> Warning: This library has not even been proven to work correctly and is not being maintained. Use at your own risk.

GoLoki is a Go library that allows programs to send log messages to Loki, a component of Grafana. Log messages are sent as [JSON data in HTTP POST requests](https://github.com/grafana/loki/blob/main/docs/sources/reference/api.md#ingest-logs). Logs may include structured metadata which is all metadata that does not match a key in `logger.MetadataLabels`.

## Usage

```go
package main

import "github.com/merlinz01/goloki"

func main() {
    logger := goloki.LokiLogger{}
    logger.LokiUrl = "http://localhost:3100/loki/api/v1/push"
    logger.MetadataLabels = append(logger.MetadataLabels, "path")
    logger.MetadataLabels = append(logger.MetadataLabels, "statuscode")

    logger.Setup()

    message := goloki.NewMessage("192.168.1.1 GET /index.html -> 200 (295 B)")
    message.Metadata["client"] = "192.168.1.1"
    message.Metadata["method"] = "GET"
    message.Metadata["path"] = "/index.html"
    message.Metadata["statuscode"] = "200"
    message.Metadata["size"] = "295"

    logger.Log(&message)
}
```
