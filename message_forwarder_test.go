package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/Financial-Times/go-logger/v2"
	producer "github.com/Financial-Times/message-queue-go-producer/producer"
	consumer "github.com/Financial-Times/message-queue-gonsumer"
)

func TestExtractTID(t *testing.T) {
	var tests = []struct {
		msg                   consumer.Message
		expectedTransactionID string
		expectedErrorMsg      string
	}{
		{consumer.Message{
			Headers: map[string]string{
				"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
				"Message-Timestamp": "2015-07-06T07:03:09.362Z",
				"Message-Type":      "cms-content-published",
				"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
				"Content-Type":      "application/json",
				"X-Request-Id":      "tid_t9happe59y",
			},
			Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"tid_t9happe59y",
			"",
		},
		{
			consumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			`header "X-Request-Id" is empty or missing`,
		},
		{
			consumer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"t9happe59y",
			"",
		},
	}

	for _, test := range tests {
		actualTransactionID, err := extractTID(test.msg.Headers)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedTransactionID != actualTransactionID {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedTransactionID, actualTransactionID)
		}
	}
}

// Tests that both enrichment and the check if a message is forwardable are performed in forwardMsg
func TestForwardMsg(t *testing.T) {
	logger := log.NewUPPLogger("Test", "PANIC")

	var tests = []struct {
		name        string
		region      string
		msg         consumer.Message
		expectedMsg producer.Message
	}{
		{
			name:        "Message is skipped successfully",
			region:      "eu",
			msg:         buildConsumerMessageWithRegion("eu"),
			expectedMsg: producer.Message{},
		},
		{
			name:   "Origin-Region header is enriched successfully",
			region: "eu",
			msg:    buildConsumerMessageWithRegion(""),
			expectedMsg: producer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Origin-Region":     "us",
					"Content-Type":      "application/json",
					"X-Request-Id":      "tid_t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &MockMessageProducer{}
			app := &BridgeApp{
				producerInstance: p,
				logger:           logger,
				region:           test.region,
			}

			app.forwardMsg(test.msg)
			assert.Equal(t, test.expectedMsg, p.Msg)
		})
	}
}

type MockMessageProducer struct {
	Msg producer.Message
}

func (p *MockMessageProducer) SendMessage(uuid string, msg producer.Message) error {
	p.Msg = msg
	return nil
}

func (p MockMessageProducer) ConnectivityCheck() (string, error) {
	return "Connectivity is OK", nil
}

func TestIsMsgForwardable(t *testing.T) {
	var tests = []struct {
		name     string
		region   string
		msg      consumer.Message
		expected bool
	}{
		{
			name:     "Service without region forwards messages with no Origin-Region header",
			region:   "",
			msg:      buildConsumerMessageWithRegion(""),
			expected: true,
		},
		{
			name:     "Service without region forwards all regional messages",
			region:   "",
			msg:      buildConsumerMessageWithRegion("us"),
			expected: true,
		},
		{
			name:     "Same-region message to be skipped",
			region:   "eu",
			msg:      buildConsumerMessageWithRegion("eu"),
			expected: false,
		},
		{
			name:     "Different-region message to be forwarded",
			region:   "eu",
			msg:      buildConsumerMessageWithRegion("us"),
			expected: true,
		},
		{
			name:     "Message without Origin-Region header to be forwarded",
			region:   "eu",
			msg:      buildConsumerMessageWithRegion(""),
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, isMsgForwardable(test.msg, test.region))
		})
	}
}

func TestEnrichMsg(t *testing.T) {
	logger := log.NewUPPLogger("Test", "PANIC")

	var tests = []struct {
		name              string
		msg               consumer.Message
		appRegion         string
		expectedMsgRegion string
	}{
		{
			name:              `EU app writes messages with "us" region`,
			msg:               buildConsumerMessageWithRegion(""),
			appRegion:         "eu",
			expectedMsgRegion: "us",
		},
		{
			name:              `US app writes messages with "eu" region`,
			msg:               buildConsumerMessageWithRegion(""),
			appRegion:         "us",
			expectedMsgRegion: "eu",
		},
		{
			name:              `App with no region does not add "Origin-Region" header`,
			msg:               buildConsumerMessageWithRegion(""),
			appRegion:         "",
			expectedMsgRegion: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enrichMsg(&test.msg, test.appRegion, logger)
			assert.Equal(t, test.expectedMsgRegion, test.msg.Headers["Origin-Region"])
		})
	}

	t.Run("Transaction ID added to message header", func(t *testing.T) {
		msg := &consumer.Message{
			Headers: map[string]string{
				"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
				"Message-Timestamp": "2015-07-06T07:03:09.362Z",
				"Message-Type":      "cms-content-published",
				"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
				"Content-Type":      "application/json",
			},
			Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`}
		enrichMsg(msg, "", logger)

		var validID = regexp.MustCompile(`^tid_.+_kafka_bridge$`)
		assert.True(t, validID.MatchString(msg.Headers["X-Request-Id"]))
	})
}

func buildConsumerMessageWithRegion(region string) consumer.Message {
	msg := consumer.Message{
		Headers: map[string]string{
			"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
			"Message-Timestamp": "2015-07-06T07:03:09.362Z",
			"Message-Type":      "cms-content-published",
			"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
			"Content-Type":      "application/json",
			"X-Request-Id":      "tid_t9happe59y",
		},
		Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`}

	if region != "" {
		msg.Headers["Origin-Region"] = region
	}
	return msg
}
