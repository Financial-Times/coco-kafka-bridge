package main

import (
	"strings"
	"testing"
)

func TestExtractJSON(t *testing.T) {

	var tests = []struct {
		kafkaMsg            string
		expectedJSONContent string
	}{
		{
			`
            FTMSG/1.0
            Message-Id: bb07b9ab-0ff6-4853-bdd1-104906d7d282
            Message-Timestamp: 2015-06-17T12:16:39.022Z
            Message-Type: cms-content-published
            Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
            Content-Type: application/json
            X-Request-Id: tid_6y3oogjqhk

            { "uuid":"f9d6eecc-14b4-11e5-973e-a0f360779259","type":"EOM::CompoundStory","value":"bodor_kafka_bridge_test","attributes":[],"linkedObjects":[] }
            `,
			`{ "uuid":"f9d6eecc-14b4-11e5-973e-a0f360779259","type":"EOM::CompoundStory","value":"bodor_kafka_bridge_test","attributes":[],"linkedObjects":[] }`,
		},
	}

	for _, test := range tests {
		actualJSONContent, err := extractJSON(test.kafkaMsg)
		if err != nil || test.expectedJSONContent != actualJSONContent {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedJSONContent, actualJSONContent)
		}
	}
}

func TestBuildHTTPEndpoint(t *testing.T) {
	var tests = []struct {
		host, expectedHttpEndpoint string
	}{
		{
			"123-cluster-elb-456.eu-west-1.elb.amazonaws.com",
			"http://123-cluster-elb-456.eu-west-1.elb.amazonaws.com/notify",
		},
		{
			"/123-cluster-elb-456.eu-west-1.elb.amazonaws.com/",
			"http://123-cluster-elb-456.eu-west-1.elb.amazonaws.com/notify",
		},
	}

	for _, test := range tests {
		actualHTTPEndpoint := buildHttpEndpoint(test.host)
		if test.expectedHttpEndpoint != actualHTTPEndpoint {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedHttpEndpoint, actualHTTPEndpoint)
		}
	}
}

func TestExtractTID(t *testing.T) {
	var tests = []struct {
		msg                   string
		expectedTransactionId string
		expectedErrorMsg      string
	}{
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json
			X-Request-Id: tid_t9happe59y

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"tid_t9happe59y",
			"",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"",
			"X-Request-Id header could not be found",
		},
		{
			`
			Message-Id: fc429b46-2500-4fe7-88bb-fd507fbaf00c
			Message-Timestamp: 2015-07-06T07:03:09.362Z
			Message-Type: cms-content-published
			Origin-System-Id: http://cmdb.ft.com/systems/methode-web-pub
			Content-Type: application/json
			X-Request-Id: t9happe59y

			{"uuid":"7543220a-2389-11e5-bd83-71cb60e8f08c","type":"EOM::CompoundStory","value":"test"}
			`,
			"",
			"Transaction id is not in expected format.",
		},
	}

	for _, test := range tests {
		actualTransactionId, err := extractTID(test.msg)
		if err != nil && !strings.Contains(err.Error(), test.expectedErrorMsg) {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedErrorMsg, err.Error())
		}
		if err == nil && test.expectedTransactionId != actualTransactionId {
			t.Errorf("\nExpected: %s\nActual: %s", test.expectedTransactionId, actualTransactionId)
		}
	}
}
