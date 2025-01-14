# Concurrent Web Server in Go

Go 1.22.2 or later installed.

### Run

   ```bash
   go run main.go
   ```
By default, the server will start at `http://localhost:4000`.

### Endpoints:

- **POST /data**:
  Accepts JSON data to store in the in-memory database.
  ```bash
  curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' http://localhost:4000/data
  ```

- **GET /data**:
  Retrieves the entire in-memory database.
  ```bash
  curl http://localhost:4000/data
  ```

- **GET /stats**:
  Returns the total number of requests handled by the server.
  ```bash
  curl http://localhost:4000/stats
  ```

- **DELETE /data/{key}**:
  Deletes a specific key from the database.
  ```bash
  curl -X DELETE http://localhost:4000/data/key
  ```
  