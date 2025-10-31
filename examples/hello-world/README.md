# Hello World Example

This is a simple example application used for testing cloud-deploy deployments.

## What's Inside

- **Dockerfile**: A simple Docker container based on nginx:alpine
- **index.html**: A basic HTML page that displays "Hello from cloud-deploy!"

## Purpose

This example is used for:
- Integration testing of cloud providers
- Quick deployment verification
- Documentation examples

## How to Use

### Local Testing

Build and run the Docker container locally:

```bash
docker build -t hello-world .
docker run -p 8080:80 hello-world
```

Open http://localhost:8080 in your browser.

### Deploy with cloud-deploy

This example is automatically used by integration tests. You can also deploy it manually:

#### AWS Example

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-1

application:
  name: hello-world
  description: "Simple hello world application"

environment:
  name: hello-world-env
  cname: my-hello-world

deployment:
  platform: docker
  source:
    type: local
    path: ./examples/hello-world

instance:
  type: t3.micro
  environment_type: SingleInstance

health_check:
  type: basic
  path: /
```

Deploy:

```bash
cloud-deploy -manifest hello-world-aws.yaml -command deploy
```

#### GCP Example

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: your-project-id
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  credentials:
    service_account_key_path: /path/to/key.json

application:
  name: hello-world
  description: "Simple hello world application"

environment:
  name: hello-world-env

deployment:
  platform: docker
  source:
    type: local
    path: ./examples/hello-world

instance:
  type: cloud-run
  environment_type: serverless

cloud_run:
  cpu: "1"
  memory: "512Mi"
  max_concurrency: 80

health_check:
  type: basic
  path: /
```

Deploy:

```bash
cloud-deploy -manifest hello-world-gcp.yaml -command deploy
```

## Cleanup

After testing, destroy the deployment:

```bash
cloud-deploy -manifest <your-manifest>.yaml -command destroy
```
