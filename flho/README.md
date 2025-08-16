# Flho

Flho is a workflow orchestration service that manages the execution of multi-step workflows with automatic retry mechanisms. It is designed to handle long-running workflows where each step may fail and require retries after specified intervals.

## Features

- Asynchronous workflow execution
- Automatic retry mechanisms with configurable intervals
- Persistent state management
- HTTP-based retry notifications
- Web-based UI for viewing workflow runs
- Workflow run tracking

## Getting Started

### Prerequisites

- Go 1.22 or newer
- Docker (optional)

### Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/windevkay/forge.git
   ```
2. Navigate to the `flho` directory:
   ```sh
   cd forge/flho
   ```
3. Install dependencies:
   ```sh
   make deps
   ```

### Running the Application

You can run the application using either `go run` or Docker.

#### Using `go run`

To run the application, you need to provide a path to a workflow configuration file. A sample configuration is provided in `sample.yml`.

```sh
make run WORKFLOWS=sample.yml
```

By default, the application will be available at `http://localhost:4000`.

#### Using Docker

To run the application using Docker, you first need to build the Docker image:

```sh
make docker-build
```

Then, you can run the Docker container:

```sh
make docker-run WORKFLOWS=sample.yml
```

## API Endpoints & UI

The following are the available API endpoints:

- `POST /initiateWorkflow`: Initiates a new workflow.
- `POST /updateWorkflowRun`: Updates the current step of a workflow.
- `POST /completeWorkflowRun`: Marks a workflow as complete.
- `GET /health`: Checks the health of the application.

### Web UI

- `GET /runs`: Provides a web interface to view all workflow runs. This endpoint is accessible via a web browser and allows you to see the status of each workflow, including ongoing, completed, and failed runs. You can filter the results by status and workflow name.

### Initiate a Workflow

To initiate a workflow, send a POST request to the `/initiateWorkflow` endpoint with the following JSON body:

```json
{
  "name": "workflow1"
}
```

This will start the workflow named `workflow1` and return a `run_id`.

### Update a Workflow

To update a workflow, send a POST request to the `/updateWorkflowRun` endpoint with the following JSON body:

```json
{
  "run_id": "your_run_id"
}
```

This will move the workflow to the next step.

### Complete a Workflow

To complete a workflow, send a POST request to the `/completeWorkflowRun` endpoint with the following JSON body:

```json
{
  "run_id": "your_run_id"
}
```

This will mark the workflow as complete.

## Workflow Configuration

Workflows are defined in a YAML file. The file should have the following structure:

```yaml
workflows:
  workflow1:
    - step0:
        name: "First Step"
        retryafter: "5s"
        retryurl: "https://example.com/retry"
    - step1:
        name: "Second Step"
        retryafter: "10s"
        retryurl: "https://example.com/retry2"
```

Each workflow is a list of steps. Each step has a name, a retry after duration, and a retry URL. When a step is executed, it will send a POST request to the retry URL with a JSON body containing the workflow name, step name, and run ID.

