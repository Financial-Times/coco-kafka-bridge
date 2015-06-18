package main

import (
    "testing"
)

func TestExtractJSON(t *testing.T) {

    var tests = []struct {
        kafkaMsg    string
        expectedJsonContent string
    } {
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
        actualJSONContent := extractJSON(test.kafkaMsg);
        if (test.expectedJsonContent != actualJSONContent) {
            t.Errorf("not equal")
        }
    }
}
