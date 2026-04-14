# Build Your First Feature (End-to-End)

This guide walks you through adding a real feature to Vein:
- create a new `tasks` resource
- add DB migration
- add data model methods
- add handlers and routes
- protect routes with auth/roles
- test the feature

## What We Will Build
Endpoints:
- `POST /v1/tasks` (admin/manager)
- `GET /v1/tasks` (admin/manager)

Task fields:
- `id` (UUID from DB)
- `title`
- `status` (`pending|done`)
- `created_at`

## 1. Create Migration
Generate migration files:

```bash
make migrate-create name=create_tasks
```

Edit the new `up` file:

```sql
CREATE TABLE IF NOT EXISTS tasks (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title VARCHAR(255) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
```

Edit the new `down` file:

```sql
DROP TABLE IF EXISTS tasks;
```

Apply migration:

```bash
make migrate-up
```

## 2. Add Data Layer
Create file: `internal/data/tasks.go`

```go
package data

import (
	"context"
	"time"

	"github.com/ebitezion/vein/internal/validator"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskModel struct {
	DB DBTX
}

func ValidateTask(v *validator.Validator, task *Task) {
	v.Check(task.Title != "", "title", "must be provided")
	v.Check(len(task.Title) <= 255, "title", "must not exceed 255 characters")
	v.Check(validator.In(task.Status, "pending", "done"), "status", "must be pending or done")
}

func (m TaskModel) Insert(task *Task) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO tasks (title, status) VALUES ($1, $2) RETURNING id, created_at`
	return m.DB.QueryRowContext(ctx, query, task.Title, task.Status).Scan(&task.ID, &task.CreatedAt)
}

func (m TaskModel) List(filters Filters) ([]Task, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELECT count(*) OVER(), id, title, status, created_at
		FROM tasks
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := m.DB.QueryContext(ctx, query, filters.PageSize, (filters.Page-1)*filters.PageSize)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	tasks := []Task{}
	total := 0
	for rows.Next() {
		var task Task
		if err := rows.Scan(&total, &task.ID, &task.Title, &task.Status, &task.CreatedAt); err != nil {
			return nil, Metadata{}, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	return tasks, CalculateMetadata(total, filters.Page, filters.PageSize), nil
}
```

## 3. Register Model In `Models`
Update `internal/data/models.go`:

```go
type Models struct {
	Users UserModel
	Tx    TxManager
	Tasks TaskModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users: UserModel{DB: db},
		Tx:    TxManager{DB: db},
		Tasks: TaskModel{DB: db},
	}
}
```

## 4. Add Handlers
Create file: `cmd/api/tasks.go`

```go
package main

import (
	"net/http"

	"github.com/ebitezion/vein/internal/data"
	"github.com/ebitezion/vein/internal/validator"
)

func (app *application) createTask(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	task := data.Task{Title: input.Title, Status: input.Status}
	if task.Status == "" {
		task.Status = "pending"
	}

	v := validator.New()
	data.ValidateTask(v, &task)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.model.Tasks.Insert(&task); err != nil {
		app.serverErrorResponse(w, r)
		return
	}

	_ = app.writeJSON(w, http.StatusCreated, envelope{"task": task}, nil)
}

func (app *application) listTasks(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	v := validator.New()

	filters := data.Filters{
		Page:         app.readInt(qs, "page", 1, v),
		PageSize:     app.readInt(qs, "page_size", 20, v),
		Sort:         "-created_at",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	data.ValidateFilters(v, filters)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	tasks, metadata, err := app.model.Tasks.List(filters)
	if err != nil {
		app.serverErrorResponse(w, r)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, envelope{"tasks": tasks, "metadata": metadata}, nil)
}
```

## 5. Add Routes
Update `cmd/api/routes.go` and register routes under auth+roles:

```go
routes.Handler(http.MethodPost, "/v1/tasks", app.authenticate(app.requireRoles("admin", "manager")(http.HandlerFunc(app.createTask))))
routes.Handler(http.MethodGet, "/v1/tasks", app.authenticate(app.requireRoles("admin", "manager")(http.HandlerFunc(app.listTasks))))
```

## 6. Run App And Try Feature
Start server:

```bash
make run
```

Issue token:

```bash
TOKEN=$(curl -s -X POST http://localhost:4000/v1/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@vein.dev","password":"VeinPass#2026!"}' | jq -r '.auth.token')
```

Create task:

```bash
curl -X POST http://localhost:4000/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write first feature tutorial","status":"pending"}'
```

List tasks:

```bash
curl 'http://localhost:4000/v1/tasks?page=1&page_size=10' \
  -H "Authorization: Bearer $TOKEN"
```

## 7. Add Tests
### 7.1 Unit tests
Create `cmd/api/tasks_test.go` and test:
- validation failure when title missing
- success on create
- list returns `tasks` and `metadata`

### 7.2 Integration test
Extend `cmd/api/e2e_test.go`:
- apply tasks migration
- create token
- call `POST /v1/tasks`
- assert `201`
- call `GET /v1/tasks`
- assert created task appears

Run tests:

```bash
make test
make test-integration
```

## 8. Quality And Safety Checks
Before pushing:

```bash
make fmt-check
make migration-check
make vet
make lint
```

## 9. Common Mistakes
- forgetting to register model in `NewModels`
- adding route without auth wrapper
- missing `SortSafelist` in filtered list endpoints
- skipping validation for input structs
- using DB operations without timeout context

## 10. Pattern To Reuse For Future Features
Repeat this order every time:
1. migration
2. data model methods
3. handlers
4. routes
5. unit tests
6. e2e tests
7. quality checks

That sequence keeps your feature consistent with the framework style and production standards.
