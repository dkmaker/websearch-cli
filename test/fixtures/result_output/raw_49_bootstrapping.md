---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 6
cached: false
---

# Setting Up a TypeScript AWS CDK Project with Multiple Stacks

## Project Structure and Initial Setup

Start by creating a CDK project using the CDK CLI, then structure it with multiple stacks that can share custom constructs. The basic pattern involves defining a base stack class with configurable properties, then instantiating multiple stack instances in your application file.

## Creating Multiple Stacks with Shared Constructs

**Define a reusable stack class** that accepts properties to configure resource behavior:

```typescript
import { Stack, StackProps } from 'aws-cdk-lib';
import { Construct } from 'constructs';

interface CustomStackProps extends StackProps {
  environment: string;
  encryptResources: boolean;
}

export class CustomStack extends Stack {
  constructor(scope: Construct, id: string, props: CustomStackProps) {
    super(scope, id, props);
    // Use props to conditionally create resources
  }
}
```

**Create shared custom constructs** in a separate directory (e.g., `lib/constructs/`) that can be imported and reused across stacks:

```typescript
export class SharedConstruct extends Construct {
  constructor(scope: Construct, id: string, props?: any) {
    super(scope, id);
    // Define reusable resources
  }
}
```

**Instantiate multiple stacks** in your application file, each using the shared constructs:

```typescript
import * as cdk from 'aws-cdk-lib';
import { CustomStack } from '../lib/stacks/custom-stack';

const app = new cdk.App();

new CustomStack(app, 'ProdStack', {
  environment: 'prod',
  encryptResources: true,
  env: { region: 'us-east-1' }
});

new CustomStack(app, 'DevStack', {
  environment: 'dev',
  encryptResources: false,
  env: { region: 'us-west-2' }
});
```

## Testing Each Stack with Jest and CDK Assertions

Create test files in a `tests/` directory for each stack:

```typescript
import { Template } from 'aws-cdk-lib/assertions';
import * as cdk from 'aws-cdk-lib';
import { CustomStack } from '../lib/stacks/custom-stack';

describe('ProdStack', () => {
  test('creates S3 bucket with encryption', () => {
    const app = new cdk.App();
    const stack = new CustomStack(app, 'TestStack', {
      environment: 'prod',
      encryptResources: true
    });

    const template = Template.fromStack(stack);
    template.hasResourceProperties('AWS::S3::Bucket', {
      BucketEncryption: {
        ServerSideEncryptionConfiguration: [{}]
      }
    });
  });
});
```

Configure `jest.config.js` in your project root:

```javascript
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: ['**/*.test.ts'],
  collectCoverageFrom: [
    'lib/**/*.ts',
    '!lib/**/*.d.ts'
  ]
};
```

## Parallel Synth in CI

For parallel synthesis during CI/CD, use the AWS CDK CLI's capability to deploy individual stacks. Configure your CI pipeline (GitHub Actions, CodePipeline, etc.) to synth stacks in parallel:

```yaml
# GitHub Actions example
jobs:
  synth:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        stack: ['ProdStack', 'DevStack']
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm install
      - run: npm test
      - run: npx cdk synth ${{ matrix.stack }}
```

Use `stack.addDependency(stack)` to define explicit deployment order if some stacks depend on others:

```typescript
devStack.addDependency(prodStack);
```

**Note:** The search results cover multiple stacks and nested stacks architecture but don't include detailed testing and CI patterns. The testing and CI examples above follow AWS CDK best practices but represent supplemental guidance beyond the provided documentation.