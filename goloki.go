package goloki

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type LokiMessage struct {
	timestamp time.Time
	raw       string
	Metadata  map[string]string
}

func NewMessage(raw string) LokiMessage {
	res := LokiMessage{raw: raw}
	res.Metadata = make(map[string]string)
	return res
}

type LokiLogger struct {
	httpClient     http.Client
	LokiUrl        string
	MetadataLabels []string
	messages       chan *LokiMessage
	QueueLength    uint32
	MaxQueueTime   time.Duration
	quit           chan struct{}
	queue          []*LokiMessage
	waitgroup      sync.WaitGroup
}

func (logger *LokiLogger) Setup() {
	if logger.LokiUrl == "" {
		panic("no url specified")
	}
	if logger.QueueLength == 0 {
		logger.QueueLength = 100
	}
	if logger.MaxQueueTime == 0 {
		logger.MaxQueueTime = time.Second * 10
	}
	logger.messages = make(chan *LokiMessage)
	logger.quit = make(chan struct{})
	logger.waitgroup.Add(1)
	go logger.run()
}

func (logger *LokiLogger) Shutdown() {
	close(logger.quit)
	logger.waitgroup.Wait()
}

func (logger *LokiLogger) Log(message *LokiMessage) {
	message.timestamp = time.Now()
	logger.messages <- message
}

func (logger *LokiLogger) run() {
	defer func() {
		if len(logger.queue) > 0 {
			logger.sendQueue()
		}
		logger.waitgroup.Done()
	}()
	maxWait := time.NewTimer(logger.MaxQueueTime)
	for {
		select {
		case <-logger.quit:
			return
		case message := <-logger.messages:
			logger.queue = append(logger.queue, message)
			if len(logger.queue) >= int(logger.QueueLength) {
				logger.sendQueue()
				maxWait.Reset(logger.MaxQueueTime)
			}
		case <-maxWait.C:
			if len(logger.queue) > 0 {
				logger.sendQueue()
				maxWait.Reset(logger.MaxQueueTime)
			}
		}
	}
}

func (logger *LokiLogger) sendQueue() {
	log.Println("sending queue")
	body := new(bytes.Buffer)
	body.WriteString("{\"streams\": [")
	for i, message := range logger.queue {
		body.Write(logger.formatMessageStream(message))
		if i != len(logger.queue)-1 {
			body.WriteString(", ")
		}
	}
	body.WriteString("] }\n")
	println(body.String())
	logger.queue = []*LokiMessage{}
	logger.postJsonRequest(body)
	log.Println("Sent a batch of log messages")
}

func (logger *LokiLogger) formatMessageStream(message *LokiMessage) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("{ \"stream\": ")
	var labelvalues = make(map[string]string)
	for _, label := range logger.MetadataLabels {
		labelvalues[label] = message.Metadata[label]
		delete(message.Metadata, label)
	}
	value, _ := json.Marshal(labelvalues)
	buf.Write(value)
	buf.WriteString(", \"values\": [ [ \"")
	buf.WriteString(strconv.FormatInt(message.timestamp.Unix()*1_000_000_000, 10))
	buf.WriteString("\", ")
	value, _ = json.Marshal(message.raw)
	buf.Write(value)
	buf.WriteString(", ")
	value, _ = json.Marshal(message.Metadata)
	buf.Write(value)
	buf.WriteString(" ] ] } ")
	return buf.Bytes()
}

func (logger *LokiLogger) postJsonRequest(body io.Reader) (err error) {
	request, err := http.NewRequest("POST", logger.LokiUrl, body)
	if err != nil {
		log.Println("Failed to create new HTTP request for log data")
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := logger.httpClient.Do(request)
	if err != nil {
		log.Println("Failed to post JSON log data")
		return
	}
	rb, _ := io.ReadAll(response.Body)
	log.Println(string(rb))
	defer response.Body.Close()
	if response.StatusCode != 204 {
		log.Println("Unexpected HTTP status while transferring log data:", response.StatusCode)
		return UnexpectedHTTPResult{"Unexpected HTTP result"}
	}
	return nil
}

type UnexpectedHTTPResult struct {
	s string
}

func (err UnexpectedHTTPResult) Error() string {
	return err.s
}
