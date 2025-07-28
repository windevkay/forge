# Flho Workflow Runs UI

This directory contains the HTML templates for the Flho workflow management system.

## Features

### `/runs` Endpoint

The runs page provides a web interface to view and filter workflow runs with the following features:

- **View all workflow runs** with their current status, step information, and timing
- **Filter by status**: ongoing, completed, or failed
- **Search by workflow name** using partial text matching
- **Pagination** with 20 items per page by default
- **Responsive design** using Bootstrap 5
- **Real-time refresh** via manual refresh button

### Available Filters

- `status`: Filter by run status (`ongoing`, `completed`, `failed`)
- `workflow`: Search by workflow name (partial match)
- `page`: Page number for pagination (default: 1)
- `pageSize`: Items per page (default: 20)

### Example URLs

- View all runs: `http://localhost:4000/runs`
- View ongoing runs: `http://localhost:4000/runs?status=ongoing`
- Search for specific workflow: `http://localhost:4000/runs?workflow=user_onboarding`
- Combined filters: `http://localhost:4000/runs?status=failed&workflow=payment`

### Template Structure

- `runs.html`: Main template for the runs listing page
- Uses Bootstrap 5 for styling and responsive layout
- Includes helper functions for formatting times and durations
- Features pagination controls with proper URL parameter handling
