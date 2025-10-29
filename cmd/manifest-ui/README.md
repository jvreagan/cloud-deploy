# Cloud Deploy Manifest UI

Web-based interface for generating cloud-deploy manifest files.

## Overview

The Manifest UI provides a simple, form-based interface for creating deployment manifests for AWS and GCP. It eliminates the need to manually write YAML files and helps prevent configuration errors.

## Features

- **Provider Selection**: Choose between AWS (Elastic Beanstalk) and GCP (Cloud Run)
- **Dynamic Forms**: Form fields change based on the selected provider
- **Required/Optional Fields**: Clear indication of which fields are required
- **Validation**: Client and server-side validation
- **Auto-generation**: Generates properly formatted YAML manifests
- **File Storage**: Saves manifests to `generated-manifests/` directory with timestamps

## Running the Server

### From the manifest-ui directory:

```bash
cd cmd/manifest-ui
go run main.go
```

### From the project root:

```bash
go run cmd/manifest-ui/main.go
```

The server will start on `http://localhost:5001`

## Using the UI

1. **Open your browser** and navigate to `http://localhost:5001`

2. **Select a provider** (AWS or GCP)

3. **Fill in the form**:
   - Fields marked with a red asterisk (*) are required
   - Help text is provided for most fields
   - Form fields automatically show/hide based on selections

4. **Generate manifest**:
   - Click "Generate Manifest" button
   - Success message will show the file path
   - Manifest is saved to `generated-manifests/` directory

5. **Use the manifest**:
   ```bash
   cloud-deploy -command deploy -manifest generated-manifests/aws-manifest-20241029-123456.yaml
   ```

## Form Sections

### Common (All Providers)
- **Application**: Name and description
- **Environment**: Environment name and settings
- **Deployment**: Platform and source configuration
- **Environment Variables**: Key-value pairs (JSON format)
- **Tags**: Resource tags (JSON format)

### AWS-Specific
- **AWS Configuration**: Region, solution stack
- **Instance Configuration**: Instance type, environment type
- **Health Check**: Type and path
- **Monitoring**: Enhanced health, CloudWatch metrics, logs
- **IAM**: Instance profile
- **Credentials**: Optional AWS credentials (use AWS CLI credentials if omitted)

### GCP-Specific
- **GCP Configuration**: Region, project ID, billing account ID
- **Cloud Run Configuration**: CPU, memory, scaling, concurrency, timeout
- **Credentials**: Service account key path (required)
- **Monitoring**: Cloud Logging configuration

## Generated Files

Manifests are saved with the following naming pattern:
- `{provider}-manifest-{timestamp}.yaml`
- Example: `aws-manifest-20241029-143022.yaml`

Files are stored in: `generated-manifests/` (created automatically if it doesn't exist)

## Testing

Run the test suite:

```bash
cd cmd/manifest-ui
go test -v
```

Test coverage includes:
- AWS manifest generation
- GCP manifest generation
- Invalid request handling
- Minimal configuration handling
- YAML format validation

## API Endpoint

The server exposes one API endpoint:

### POST /api/generate

Generates a manifest file from JSON input.

**Request Body**: JSON object matching the `ManifestRequest` structure

**Response**: JSON object with:
- `message`: Success/error message
- `filename`: Generated filename
- `path`: Full path to generated file

**Example**:
```bash
curl -X POST http://localhost:5001/api/generate \
  -H "Content-Type: application/json" \
  -d @request.json
```

## Architecture

```
cmd/manifest-ui/
├── main.go          # HTTP server and API handlers
├── main_test.go     # Test suite
└── README.md        # This file

web/static/
└── index.html       # Web interface (HTML, CSS, JavaScript)

generated-manifests/ # Output directory (created automatically)
```

## Development

The UI is built with:
- **Backend**: Go HTTP server with standard library
- **Frontend**: Plain HTML, CSS, and vanilla JavaScript
- **No dependencies**: No frontend frameworks or build tools required

This keeps the UI simple, fast, and easy to deploy.

## Future Enhancements

Planned features:
- Manifest validation before generation
- Load existing manifests for editing
- Preview generated YAML before saving
- Export/download functionality
- Azure and OCI provider support
- Template library

## Troubleshooting

**Port already in use**:
```
Error: listen tcp :5001: bind: address already in use
```
Solution: Stop other processes using port 5001 or change the port in `main.go`

**Generated file not found**:
- Check the `generated-manifests/` directory exists
- Verify write permissions in the project directory
- Check server logs for error messages

**Form validation errors**:
- Ensure all required fields (marked with *) are filled
- JSON fields must be valid JSON format
- Check help text for field format requirements
