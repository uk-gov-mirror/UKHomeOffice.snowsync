## snowsync

These AWS Lambda functions aim to create a bi-directional incident ticketing integration between ACP Service Desk and ServiceNow.

The functions are intended to cover a small set of use cases that meet ACP's specific needs, and their reusability in other circumstances is likely to be very limited. As per the accompanying license, the code in this repo is provided "as is" and without warranty of any kind. It can change without notice or any regard for backwards compatibility.

### ACP to ServiceNow
When an incident ticket is raised, a webhook is sent from ACP Service Desk to AWS API Gateway which triggers the [outbound function](./pkg/out). The payload is parsed and forward to ServiceNow which replicates the ticket and returns an identifier. This is written to AWS DynamoDB along with the original ticket details. 

Further updates to tickets are made using the returned identifier. The database is checked at every transmission to find first a partial and then an exact match using ticket and comment identifiers. 

If no match is found, ticket creation workflow is triggered. If a partial match is found, comment update workflow is triggered. If an exact match is found, only ticket progress is updated.  

### ServiceNow to ACP
Inversely, a webhook from ServiceNow triggers the [inbound function](./pkg/in) which parses the payload and forwards it to ACP Service Desk. The identifier in response is written to AWS DynamoDB along with the original ticket details. 

Further updates are made using the ACP provided identifier following the same workflow logic as above.

### Deployment
Terraform resources (acp-lambda-snowsync) can be found in ACP Gitlab.