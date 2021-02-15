## snowsync

[![Go Report Card](https://goreportcard.com/badge/github.com/UKHomeOffice/SNowsync)](https://goreportcard.com/report/github.com/UKHomeOffice/snowsync)

These AWS Lambda functions aim to create a bi-directional incident ticketing integration between ACP Service Desk and ServiceNow.

The functions are intended to cover a small set of use cases that meet ACP's specific needs, and their reusability in other circumstances is likely to be very limited. As per the accompanying license, the code in this repo is provided "as is" and without warranty of any kind. It can change without notice or any regard for backwards compatibility.

### ACP to ServiceNow
When an incident ticket is raised, a webhook is sent from ACP Service Desk to AWS API Gateway which triggers [outapi](./pkg/outapi). The payload is parsed and written to AWS SQS which then triggers [outprocessor](./pkg/outprocessor). The incident is forwarded by [outcaller](./pkg/outcaller) on to ServiceNow which replicates the ticket and returns an identifier. This is written to AWS DynamoDB along with the original ticket details. 

Subsequent updates to tickets are made using the returned identifier. The database is checked at every transmission to find first a partial and then an exact match using ticket and comment identifiers. If no match is found, ticket creation workflow is triggered. If a partial match is found, comment update workflow is triggered. If an exact match is found, only ticket progress is updated.  

### ServiceNow to ACP
Similarly, webhooks from ServiceNow trigger [inapi](./pkg/inapi) which parses the attached payload. The values are not written to SQS in this direction because of the need to return the ACP identifier in HTTP response. The ticket details are forwarded by [inprocessor](./pkg/inprocessor) to ACP Service Desk and written to DynamoDB. Further updates are made using the ACP provided identifier following the same workflow logic as above.

### Deployment
Terraform resources (acp-lambda-snowsync) can be found in ACP Gitlab.