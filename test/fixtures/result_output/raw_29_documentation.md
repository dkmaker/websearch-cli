---
provider: brave
mode: web
mode_adjusted: true
truncated: false
sources_count: 5
cached: false
---

**AWS CDK Custom Resources — AWS Cloud Development Kit 2.238.0 documentation**
When CloudFormation needs to create, update or delete a custom resource, **it sends a lifecycle event notification to a custom resource provider**. The provider handles the event (e.g. creates a resource) and sends back a response to CloudFormation. The @aws-cdk/custom-resources.Provider construct ...

**aws-cdk-lib.custom_resources module · AWS CDK**
When CloudFormation needs to create, update or delete a custom resource, **it sends a lifecycle event notification to a custom resource provider**. The provider handles the event (e.g. creates a resource) and sends back a response to CloudFormation.

**Introduction to Using Custom Resources in AWS CDK — Automating your infrastructure | by Juin | Medium**
The onEventHandler should also return the following parameters: **PhysicalResourceId is the resource ID of the resource that is created as defined by your Lambda function**. You should take care when updating this as updating this during a Update ...

**aws-cdk.custom-resources · PyPI**
When CloudFormation needs to create, update or delete a custom resource, **it sends a lifecycle event notification to a custom resource provider**. The provider handles the event (e.g. creates a resource) and sends back a response to CloudFormation.

**ProviderProps — AWS Cloud Development Kit 2.233.0 documentation**
# Create custom resource handler entrypoint handler = lambda_.Function(self, "my-handler", runtime=lambda_.Runtime.NODEJS_20_X, handler="index.handler", code=lambda_.Code.from_inline(""" exports.handler = async (event, context) => { return { PhysicalResourceId: '1234', NoEcho: true, Data: { mySecret: 'secret-value', hello: 'world', ghToken: 'gho_xxxxxxx', }, }; };""") ) # Provision a custom resource provider framework provider = cr.Provider(self, "my-provider", on_event_handler=handler ) CustomResource(self, "my-cr", service_token=provider.service_token )