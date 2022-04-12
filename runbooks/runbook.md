<!--
    Written in the format prescribed by https://github.com/Financial-Times/runbook.md.
    Any future edits should abide by this format.
-->
# UPP - Kafka Bridge

The purpose of the Kafka Bridge is to replicate (bridge) messages from one UPP Kubernetes cluster to another.

## Code

kafka-bridge

## Primary URL

https://upp-prod-delivery-glb.upp.ft.com/__cms-kafka-bridge-pub-prod-eu/

## Service Tier

Platinum

## Lifecycle Stage

Production

## Host Platform

AWS

## Architecture

The Kafka bridges consume messages from Kafka via the Kafka REST proxy service in one K8s cluster and adds the messages in the same Kafka topic in another cluster using either the Kafka REST proxy service or the CMS Notifier service.
These bridges are located in all the UPP Delivery clusters replicating messages from the "NativeCmsPublicationEvents" and "NativeCmsMetadataPublicationEvents" Kafka topics in both EU and US publishing clusters (and thus guarantee the publish will go through regardless of its origin region).

Additionally, there are bridges in the UPP Staging Publishing and UPP Dev Publishing clusters which replicate the production traffic in order to provide consistent testing environments.

## Contains Personal Data

No

## Contains Sensitive Data

No

<!-- Placeholder - remove HTML comment markers to activate
## Can Download Personal Data
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

<!-- Placeholder - remove HTML comment markers to activate
## Can Contact Individuals
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

## Failover Architecture Type

ActiveActive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is deployed in both Delivery clusters. The failover guide for the cluster is located here:
<https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/delivery-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

It is safe to release the service without failover.

<!-- Placeholder - remove HTML comment markers to activate
## Heroku Pipeline Name
Enter descriptive text satisfying the following:
This is the name of the Heroku pipeline for this system. If you don't have a pipeline, this is the name of the app in Heroku. A pipeline is a group of Heroku apps that share the same codebase where each app in a pipeline represents the different stages in a continuous delivery workflow, i.e. staging, production.

...or delete this placeholder if not applicable to this system
-->

## Key Management Process Type

Manual

## Key Management Details

To access the service clients need to provide basic auth credentials. To rotate credentials you need to login to a particular cluster and update varnish-auth secrets.

## Monitoring

Delivery EU cluster Kafka bridges:

*   Publishing EU Content Bridge <https://upp-prod-delivery-eu.upp.ft.com/__health/__pods-health?service-name=cms-kafka-bridge-pub-prod-eu>
*   Publishing US Content Bridge <https://upp-prod-delivery-eu.upp.ft.com/__health/__pods-health?service-name=cms-kafka-bridge-pub-prod-us>
*   Publishing EU Metadata Bridge <https://upp-prod-delivery-eu.upp.ft.com/__health/__pods-health?service-name=cms-metadata-kafka-bridge-pub-prod-eu>
*   Publishing US Metadata Bridge <https://upp-prod-delivery-eu.upp.ft.com/__health/__pods-health?service-name=cms-metadata-kafka-bridge-pub-prod-us>

Delivery US cluster Kafka bridges:

*   Publishing EU Content Bridge <https://upp-prod-delivery-us.upp.ft.com/__health/__pods-health?service-name=cms-kafka-bridge-pub-prod-eu>
*   Publishing US Content Bridge <https://upp-prod-delivery-us.upp.ft.com/__health/__pods-health?service-name=cms-kafka-bridge-pub-prod-us>
*   Publishing EU Metadata Bridge <https://upp-prod-delivery-us.upp.ft.com/__health/__pods-health?service-name=cms-metadata-kafka-bridge-pub-prod-eu>
*   Publishing US Metadata Bridge <https://upp-prod-delivery-us.upp.ft.com/__health/__pods-health?service-name=cms-metadata-kafka-bridge-pub-prod-us>

Splunk alert:

*   UPP Prod - Kafka Delivery Bridges not forwarding messages: <https://financialtimes.splunkcloud.com/en-US/manager/search/saved/searches?app=&count=10&offset=0&itemType=&owner=&search=UPP%20Prod%20-%20Kafka%20Delivery%20Bridges%20not%20forwarding%20messages>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

*   Check that the Kafka REST proxy and CMS Notifier services are healthy.
*   Check that Kafka and Zookeeper are healthy.
*   Check the logs for successful processing of messages or errors.
*   Restart the service.
