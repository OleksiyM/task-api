# Task API

A RESTful API for task management with AI-powered task description generation using various AI models via [Ollama](https://ollama.ai/), [Anthropic](https://www.anthropic.com/), and [Google Gemini](https://developers.google.com/chat/gemini).

## Features

- Create, read, update, and delete tasks
- Generate task descriptions automatically using:
  - Anthropic's Claude API
  - Ollama (local AI model)
  - Google's Gemini API
- SQLite database for persistence
- Dockerized deployment

## Prerequisites

- Go 1.24+ (for local development)
- Docker and Docker Compose (for containerized deployment)
- Ollama running locally (for Ollama integration)
- API keys for Anthropic Claude and/or Google Gemini (optional)

## Installation & Setup

### Using Docker (Recommended)

1. Clone the repository
2. Set your Gemini API key in the `.env` file or through environment variables
   - Create a `.env` file with `GEMINI_API_KEY=your-key-here` (see `.env.example`)
   - This is required for the Gemini integration to work in Docker
3. Run with Docker Compose:

```bash
docker-compose up --build
```

### Local Development

1. Install Go from the [site](https://golang.org/dl/)
2. Clone the repository
3. Install dependencies:

```bash
go mod download
```

4. Set environment variables:

**Linux/Mac**

```bash
export GEMINI_API_KEY="your-api-key-here"
```

**Windows**

```bash
set GEMINI_API_KEY=your-api-key-here
```

5. Run the application:

```bash
go run main.go
```

## Testing with the Simple GUI

For quick testing, you can use the included `simple-gui.html` file:

1. Start the Task API using either Docker or local development method
2. Open the `simple-gui.html` file in any web browser
3. The GUI will allow you to perform all operations (create, read, update, delete tasks)
4. No installation is needed - simply open the HTML file from any local location

## API Endpoints

| Method | Endpoint        | Description                              |
| ------ | --------------- | ---------------------------------------- |
| GET    | `/tasks`        | Get all tasks                            |
| GET    | `/tasks/:id`    | Get a specific task                      |
| POST   | `/tasks`        | Create a task with Claude AI description |
| POST   | `/tasks/ollama` | Create a task with Ollama AI description |
| POST   | `/tasks/gemini` | Create a task with Gemini AI description |
| PUT    | `/tasks/:id`    | Update a task                            |
| DELETE | `/tasks/:id`    | Delete a task                            |

## Usage Examples

### Create a task with Gemini description

```bash
curl -X POST http://localhost:8080/tasks/gemini -H "Content-Type: application/json" -d '{"title": "Write a report"}'
```

### Create a task with Ollama description

```bash
curl -X POST http://localhost:8080/tasks/ollama -H "Content-Type: application/json" -d '{"title": "Prepare presentation"}'
```

### Get all tasks

```bash
curl http://localhost:8080/tasks
```

### Get a specific task

```bash
curl http://localhost:8080/tasks/1
```

### Update a task

```bash
curl -X PUT http://localhost:8080/tasks/1 -H "Content-Type: application/json" -d '{"title": "Updated title", "description": "Updated description"}'
```

### Delete a task

```bash
curl -X DELETE http://localhost:8080/tasks/1
```

## Configuration

For Claude integration, update the API key in the `createTask` function in `main.go`:

```go
SetHeader("Authorization", "Bearer YOUR_ANTHROPIC_API_KEY").
```

For Gemini integration, set the `GEMINI_API_KEY` environment variable.

For Ollama integration, ensure Ollama is running locally on port 11434.

## Notes

- For Claude API integration, you need to replace `YOUR_ANTHROPIC_API_KEY` with a valid API key
- The database is stored in `tasks.db` in the application directory
- CORS is enabled for all origins for easier frontend integration

## License

MIT
