package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/Financial-Times/go-logger"
	queueProducer "github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/stretchr/testify/assert"
)

func TestSendMessage(t *testing.T) {
	logger.InitDefaultLogger("kafka-bridge")
	var tests = []struct {
		config          queueProducer.MessageProducerConfig
		uuid            string
		message         queueProducer.Message
		expectedHeaders map[string]string
	}{
		{ //happy flow
			queueProducer.MessageProducerConfig{
				Addr:          "address",
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
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "authorizationkey",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"Content-Type":      "application/json",
			},
		},
		{ //missing content-type
			queueProducer.MessageProducerConfig{
				Addr:          "address",
				Authorization: "authorizationkey",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://cmdb.ft.com/systems/methode-web-pub",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "authorizationkey",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
			},
		},
		{ //authorization missing
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
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"Content-Type":      "application/json",
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
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"Content-Type":      "application/json",
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
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"Content-Type":      "application/json",
			},
		},
		{ // origin system id is invalid (but the bridge shouldn't care)
			queueProducer.MessageProducerConfig{
				Addr:          "address",
				Authorization: "authorizationkey",
			},
			"",
			queueProducer.Message{
				Headers: map[string]string{
					"Message-Id":        "fc429b46-2500-4fe7-88bb-fd507fbaf00c",
					"Message-Timestamp": "2015-07-06T07:03:09.362Z",
					"Message-Type":      "cms-content-published",
					"Origin-System-Id":  "http://foo.ft.com/systems/bar",
					"Content-Type":      "application/json",
					"X-Request-Id":      "t9happe59y",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "http://foo.ft.com/systems/bar",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "authorizationkey",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"Content-Type":      "application/json",
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
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "",
				"Message-Timestamp":  "",
				"Content-Type":      "application/json",
			},
		},
		{ //native-hash forward
			queueProducer.MessageProducerConfig{
				Addr:          "address",
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
					"Native-Hash":       "27f79e6d884acdd642d1758c4fd30d43074f8384d552d1ebb1959345",
				},
				Body: `{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}`},
			map[string]string{
				"X-Origin-System-Id": "http://cmdb.ft.com/systems/methode-web-pub",
				"X-Request-Id":       "t9happe59y",
				"Authorization":      "authorizationkey",
				"Message-Timestamp":  "2015-07-06T07:03:09.362Z",
				"X-Native-Hash":      "27f79e6d884acdd642d1758c4fd30d43074f8384d552d1ebb1959345",
				"Content-Type":      "application/json",
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
	d.assert.Equal(d.host, req.Host, "Host header value differs.")

	return &d.resp, nil
}
