## snowsync

[WIP]

[![Go Report Card](https://goreportcard.com/badge/github.com/UKHomeOffice/SNowsync)](https://goreportcard.com/report/github.com/UKHomeOffice/snowsync)

#### A suite of AWS Lambda functions to integrate ACP Service Desk with ServiceNow.

* [cmd/outapi](./cmd/outapi): Function outapi calls package outapi.
* [pkg/outapi](./pkg/outapi): Package outapi receives a webhook from Service Desk, parses its payload and writes it to SQS.

* [cmd/outprocessor](./cmd/outprocessor): Function outprocessor calls package outprocessor.
* [pkg/outprocessor](./pkg/outprocessor): Package outprocessor receives a SQS event, queries/writes to DynamoDB and invokes other functions to handle SNOW bound requests.

* [cmd/outcaller](./cmd/outcaller): Function outcaller calls package outcaller.
* [pkg/outcaller](./pkg/outcaller): Package outcaller makes HTTP requests to SNOW to create/update a ticket.
* [pkg/client](./pkg/client): Package client is a HTTP client.

* [cmd/inapi](./cmd/ianapi): Function ianapi calls package ianapi.
* [pkg/inapi](./pkg/ianapi): Package ianapi is a temporary all in one receiver function until SNOW implements ACP bound transactions.