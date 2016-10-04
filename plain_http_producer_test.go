package main

import (
	"bytes"
	"fmt"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestSendMessage(t *testing.T) {
	initLoggers()

	var tests = []struct {
		config          queueProducer.MessageProducerConfig
		uuid            string
		message         queueProducer.Message
		expectedHeaders map[string]string
	}{
		{ //happy flow
			queueProducer.MessageProducerConfig{
				Addr:          "address",
				Queue:         "kafka",
				Authorization: "authorizationkey",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "authorizationkey",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
			},
		},
		{ //authorization missing
			queueProducer.MessageProducerConfig{
				Addr:  "address",
				Queue: "kafka",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
			},
		},
		{ //host header (queue) is missing
			queueProducer.MessageProducerConfig{
				Addr: "address",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
			},
		},
		{ //origin system id is missing
			queueProducer.MessageProducerConfig{
				Addr: "address",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
			},
		},
		{ //Message-Timestamp is missing
			queueProducer.MessageProducerConfig{
				Addr: "address",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":       "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
					"Message-Type":     "cms-content-published",
					"Content-Type":     "application/json",
					"X-Request-Id":     "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
			},
		},
	}

	for _, test := range tests {
		cmsNotifierTest := &plainHTTPMessageProducer{
			test.config,
			&dummyHttpClient{
				assert:  assert.New(t),
				address: test.config.Addr,
				headers: test.expectedHeaders,
				resp: http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewBuffer([]byte{})),
				},
				host: test.config.Queue,
			},
		}
		err := cmsNotifierTest.SendMessage(test.uuid, test.message)
		if err != nil {
			t.Errorf("\nExpected error was nil \nActual: %s", err.Error())
		}
	}

}

type dummyHttpClient struct {
	assert  *assert.Assertions
	address string
	headers map[string]string
	resp    http.Response
	host    string
}

func (d *dummyHttpClient) Do(req *http.Request) (resp *http.Response, err error) {
	// Check url
	d.assert.Contains(req.URL.String(), fmt.Sprintf("%s/notify", d.address), fmt.Sprintf("Expected URL incorrect"))

	// Check that the correct headers were set
	for key, value := range d.headers {
		d.assert.Equal(value, req.Header.Get(key), fmt.Sprintf("%s Header value differs. Expected: %s, Actual: %s", key, value, req.Header.Get(key)))
	}

	// Check that host is set as expected
	d.assert.Equal(d.host, req.Host, fmt.Sprintf("%s Host header value differs."))

	return &d.resp, nil
}

func TestExtractOriginSystem(t *testing.T) {
	var tests = []struct {
		msg                  queueProducer.Message
		expectedSystemOrigin string
		expectedErrorMsg     string
	}{
		{
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"methode-web-pub",
			"",
		},
		{
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			"Origin system id is not set",
		},
		{
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			"",
			"Origin system id is not set",
		},
	}

	for _, test := range tests {
		actualSystemOrigin, err := extractOriginSystem(test.msg.Headers)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedSystemOrigin != actualSystemOrigin {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedSystemOrigin, actualSystemOrigin)
		}
	}
}
