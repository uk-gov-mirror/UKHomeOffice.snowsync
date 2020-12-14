## snowsync

[![Go Report Card](https://goreportcard.com/badge/github.com/UKHomeOffice/snowsync)](https://goreportcard.com/report/github.com/UKHomeOffice/snowsync)

#### A suite of AWS Lambda functions to integrate ACP Service Desk with SNOW.

[WIP]

* [cmd/api](./cmd/api): Function api starts a SQS session and hands over to package api.
* [pkg/api](./pkg/api): Package api receives a webhook from JSD, parses its payload and writes it to SQS.

* [cmd/processor](./cmd/processor): Function processor starts a Lambda session and hands over to package processor.
* [pkg/processor](./pkg/processor): Package processor receives a SQS event and invokes other functions to handle JSD webhooks.

* [cmd/checker](./cmd/checker): Function checker starts a DynamoDB session and hands over to package checker.
* [pkg/checker](./pkg/checker): Package checker queries DynamoDB and returns a SNOW identifier if one exists.

* [cmd/caller](./cmd/caller): Function caller hands over to package caller.
* [pkg/caller](./pkg/caller): Package caller makes a HTTP request to SNOW to create/update a ticket and returns a SNOW identifier.
* [pkg/client](./pkg/client): Package client is a HTTP client.

* [cmd/dbputter](./cmd/dbputter): Function dbputter starts a DynamoDB session and hands over to package dbputter.
* [pkg/dbputter](./pkg/dbputter): Package dbputter writes a new ticket to DynamoDB.

* [cmd/dbupdater](./cmd/dbupdater): Function dbupdater starts a DynamoDB session and hands over to package dbupdater.
* [pkg/dbupdater](./pkg/dbupdater): Package dbupdater writes a ticket update to DynamoDB.

* [cmd/receiver](./cmd/receiver): Function receiver starts a DynamoDB session and hands over to package receiver.
* [pkg/receiver](./pkg/receiver): Package receiver handles a webhook from SNOW, writes its payload to DynamoDB and makes a HTTP request to JSD to update a ticket.
