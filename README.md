# CodePulse

Asynchronous competitive programming and automated code evaluation platform (LeetCode analog) built in Go.

The platform accepts untrusted Go code submissions, queues them via Apache Kafka, compiles and executes them inside isolated, resource-constrained Docker sandboxes, stores results in PostgreSQL 18, and maintains real-time rankings and rate limiting in Redis 8.

---

## Architecture Overview

```text
                             [ HTTP Client / Browser ]
                                         │
                    ┌────────────────────┴────────────────────┐
                    │ :8080 (Traffic)                         │ :9090 (Probes)
                    ▼                                         ▼
            ┌───────────────┐                         ┌───────────────┐
            │ CodePulse API │                         │ Health Server │
            │ (REST / slog) │                         │ /healthz      │
            └───────┬───────┘                         │ /ready        │
                    │                                 └───────────────┘
          ┌─────────┼─────────────────────────┐
          │ (1)     │ (2)                     │ (3)
          ▼         ▼                         ▼
    ┌──────────┐ ┌─────────────┐        ┌───────────┐
    │ Postgres │ │ Redis 8     │        │  Kafka    │
    │    18    │ │ (RateLimit) │        │ (KRaft)   │
    └──────────┘ └─────────────┘        └─────┬─────┘
                                              │ "submissions.pending"
                                              ▼
                                    ┌──────────────────┐
                                    │ CodePulse Worker │
                                    └─────────┬────────┘
                                              │ Docker Engine API
                                              ▼
                                    ┌──────────────────┐
                                    │ Ephemeral Docker │
                                    │ Sandbox          │
                                    │ (network: none)  │
                                    │ (cgroup limits)  │
                                    └──────────────────┘
```

### Execution Flow
1. **Submission Request**: Client sends code to `POST /api/v1/submissions`.
2. **Rate Limiting**: Sliding window rate limiter checks user submission frequency (max 5 submissions/min via Redis Sorted Set). Rejects bursts with `429 Too Many Requests`.
3. **Queueing**: API stores submission with `PENDING` status in PostgreSQL and publishes an event to Kafka topic `submissions.pending`. Returns `202 Accepted` immediately.
4. **Processing**: Worker daemon consumes event, transitions status to `RUNNING`.
5. **Sandboxed Execution**: Worker spins up an isolated, disposable container (`golang:1.27-alpine`):
   - `--network none` (no external or internal network access)
   - Read-only bind mounts (`:ro`)
   - Hard cgroup memory & swap limits (e.g. 64MB)
   - CPU quota (`NanoCPUs`)
   - Execution timeout enforced via Go `context.WithTimeout`
6. **Verdict & Persistence**: Worker parses stdout/stderr and container exit code into verdicts:
   `ACCEPTED`, `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `MEMORY_LIMIT_EXCEEDED`, `COMPILATION_ERROR`, `RUNTIME_ERROR`.
7. **Scoring**: If verdict is `ACCEPTED` and the task was not previously solved by this user, points are atomically added to `contest:leaderboard` in Redis via `ZINCRBY`. Offset is committed to Kafka.

---

## Tech Stack

- **Language**: Go 1.27
- **Database**: PostgreSQL 18 (with automated `golang-migrate` schema migrations and partial B-Tree indexes)
- **Message Broker**: Apache Kafka 4.4.0 (KRaft mode, 4 partitions, partition hashing by `submission_id`)
- **Cache & Rankings**: Redis 8 (Sliding Window Log rate limiting & real-time leaderboard via `ZSET`)
- **Sandbox**: Docker Engine API (`github.com/docker/docker/client`)
- **Logging**: Structured JSON logging (`log/slog`)

---

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Ensure Docker daemon is running locally

### 1. Launch Environment
```bash
docker compose up -d --build
```
This builds and starts 6 services:
- `codepulse_postgres` (PostgreSQL 18, port 5432, with healthcheck)
- `codepulse_kafka` (Kafka 4.4 KRaft, port 9092, with healthcheck)
- `codepulse_kafka_ui` (Kafka UI, port 8085)
- `codepulse_redis` (Redis 8, port 6379, with healthcheck)
- `codepulse_api` (REST API, ports 8080 and 9090)
- `codepulse_worker` (Execution daemon)

### 2. Verify Services
```bash
docker compose ps
```
All dependency services (`postgres`, `kafka`, `redis`) should report `(healthy)`.

---

## API Reference & Usage Examples

### 1. Create a Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Sum of Two Numbers",
    "description": "Given two integers a and b, print their sum.",
    "time_limit_ms": 1000,
    "memory_limit_mb": 64,
    "points": 10,
    "tests": [
      {
        "input_data": "2 3\n",
        "expected_output": "5\n",
        "is_sample": true
      },
      {
        "input_data": "-10 15\n",
        "expected_output": "5\n",
        "is_sample": false
      }
    ]
  }'
```

### 2. List Tasks
```bash
curl http://localhost:8080/api/v1/tasks
```

### 3. Get Task Details (with sample tests)
```bash
curl http://localhost:8080/api/v1/tasks/1
```

### 4. Submit Solution (Valid A + B)
```bash
curl -i -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 1,
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "source_code": "package main\nimport \"fmt\"\nfunc main() {\n  var a, b int\n  fmt.Scan(&a, &b)\n  fmt.Println(a + b)\n}"
  }'
```
Response: `202 Accepted`
```json
{"submission_id":"da578ac0-aebd-454d-bcb8-4b691a0dbfb1"}
```

### 5. Check Submission Status & Verdict
```bash
curl http://localhost:8080/api/v1/submissions/da578ac0-aebd-454d-bcb8-4b691a0dbfb1
```

### 6. View User Submission History
```bash
curl http://localhost:8080/api/v1/users/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11/submissions
```

### 7. View Live Leaderboard
```bash
curl http://localhost:8080/api/v1/leaderboard
```
Response:
```json
[
  {
    "user_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    "score": 10
  }
]
```
*(Query param `?limit=N` supported, defaults to top 10).*

### 8. Health Checks (Port :9090)
```bash
# Liveness Probe (process is running)
curl http://localhost:9090/healthz

# Readiness Probe (checks Postgres, Redis, and Kafka connectivity)
curl http://localhost:9090/ready
```

---

## Security & Sandbox Constraints

To ensure host safety when evaluating user-submitted code:
- **No Network**: Container launched with `NetworkMode: "none"`, preventing socket connections, DNS requests, and data exfiltration.
- **Resource Constraints**:
  - `Memory`: hard-capped to task limit (e.g. 64MB).
  - `MemorySwap`: set equal to memory to strictly disallow swap usage.
  - `NanoCPUs`: throttled to prevent CPU starvation.
- **Filesystem Isolation**: Code directory is bind-mounted into `/sandbox:ro` (Read-Only) during test runs.
- **Timeout Termination**: Hard execution deadline enforced via `context.WithTimeout`. If exceeded, container is forcefully terminated and returns `TIME_LIMIT_EXCEEDED`.

---

## Running Tests

Unit tests use interface mocks and execute in-memory:
```bash
# Run service unit tests
go test -v ./internal/service/...

# Run runner sandbox tests (requires local Docker daemon)
go test -v ./internal/runner/...
```
