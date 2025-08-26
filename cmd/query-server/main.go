package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

type QueryRequest struct {
	SQL string `json:"sql"`
}

type QueryResponse struct {
	Success bool          `json:"success"`
	Data    []interface{} `json:"data,omitempty"`
	Error   string        `json:"error,omitempty"`
	Count   int           `json:"count"`
}

type QueryServer struct {
	db *sql.DB
}

func main() {
	log.Println("Starting BankAccount Transactions Query Server...")

	// Configure postgres connection using environment variables with defaults
	connectionString := getEnvWithDefault("POSTGRES_CONNECTION_STRING", "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable")
	port := getEnvWithDefault("QUERY_SERVER_PORT", "8081")

	log.Printf("Configuration:")
	log.Printf("  - Database: %s", connectionString)
	log.Printf("  - Port: %s", port)

	// Connect to database
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")

	// Create query server
	server := &QueryServer{db: db}

	// Setup Chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Add CORS headers for browser access
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Get("/health", server.healthHandler)
	r.Get("/", server.indexHandler)
	r.Post("/query", server.queryHandler)
	r.Get("/examples", server.examplesHandler)

	log.Printf("Query server starting on port %s", port)
	log.Println("Available endpoints:")
	log.Println("  GET  /health    - Health check")
	log.Println("  GET  /          - Simple web interface")
	log.Println("  POST /query     - Execute SQL query (JSON: {\"sql\": \"SELECT * FROM transactions LIMIT 10\"})")
	log.Println("  GET  /examples  - Get example queries")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *QueryServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *QueryServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>BankAccount Transactions Query Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; }
        textarea { width: 100%; height: 100px; font-family: monospace; }
        button { padding: 10px 20px; background: #007cba; color: white; border: none; cursor: pointer; }
        button:hover { background: #005a85; }
        .result { background: #f5f5f5; padding: 15px; margin-top: 20px; white-space: pre-wrap; font-family: monospace; }
        .examples { background: #e8f4fd; padding: 15px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>BankAccount Transactions Query Server</h1>
        
        <div class="examples">
            <h3>Example Queries:</h3>
            <ul>
                <li><code>SELECT * FROM transactions ORDER BY transaction_timestamp DESC LIMIT 10;</code></li>
                <li><code>SELECT account_id, owner_name, SUM(amount) as total FROM transactions GROUP BY account_id, owner_name;</code></li>
                <li><code>SELECT transaction_type, COUNT(*), SUM(amount) FROM transactions GROUP BY transaction_type;</code></li>
            </ul>
        </div>
        
        <form onsubmit="executeQuery(event)">
            <label for="sql">SQL Query:</label><br>
            <textarea id="sql" name="sql" placeholder="SELECT * FROM transactions LIMIT 10;"></textarea><br><br>
            <button type="submit">Execute Query</button>
        </form>
        
        <div id="result" class="result" style="display:none;"></div>
    </div>

    <script>
        async function executeQuery(event) {
            event.preventDefault();
            const sql = document.getElementById('sql').value;
            const resultDiv = document.getElementById('result');
            
            try {
                const response = await fetch('/query', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ sql: sql })
                });
                
                const result = await response.json();
                resultDiv.style.display = 'block';
                resultDiv.textContent = JSON.stringify(result, null, 2);
            } catch (error) {
                resultDiv.style.display = 'block';
                resultDiv.textContent = 'Error: ' + error.message;
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (s *QueryServer) examplesHandler(w http.ResponseWriter, r *http.Request) {
	examples := []map[string]string{
		{
			"name":        "Recent Transactions",
			"description": "Get the 10 most recent transactions",
			"sql":         "SELECT * FROM transactions ORDER BY transaction_timestamp DESC LIMIT 10;",
		},
		{
			"name":        "Account Balances",
			"description": "Calculate total balance per account (sum of all transactions)",
			"sql":         "SELECT account_id, owner_name, SUM(CASE WHEN transaction_type IN ('account_created', 'deposit') THEN amount ELSE -amount END) as balance FROM transactions GROUP BY account_id, owner_name ORDER BY balance DESC;",
		},
		{
			"name":        "Transaction Summary by Type",
			"description": "Count and sum transactions by type",
			"sql":         "SELECT transaction_type, COUNT(*) as count, SUM(amount) as total_amount FROM transactions GROUP BY transaction_type ORDER BY count DESC;",
		},
		{
			"name":        "Daily Transaction Volume",
			"description": "Show transaction volume by day",
			"sql":         "SELECT DATE(transaction_timestamp) as date, COUNT(*) as transactions, SUM(amount) as volume FROM transactions GROUP BY DATE(transaction_timestamp) ORDER BY date DESC;",
		},
		{
			"name":        "Large Transactions",
			"description": "Find transactions over $1000",
			"sql":         "SELECT * FROM transactions WHERE amount > 1000 ORDER BY amount DESC;",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examples)
}

func (s *QueryServer) queryHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if req.SQL == "" {
		s.sendError(w, "SQL query is required", http.StatusBadRequest)
		return
	}

	// Basic SQL injection protection - only allow SELECT statements on transactions table
	sqlLower := strings.ToLower(strings.TrimSpace(req.SQL))
	if !strings.HasPrefix(sqlLower, "select") {
		s.sendError(w, "Only SELECT queries are allowed", http.StatusBadRequest)
		return
	}

	if !strings.Contains(sqlLower, "transactions") {
		s.sendError(w, "Queries must reference the 'transactions' table", http.StatusBadRequest)
		return
	}

	// Execute query
	rows, err := s.db.Query(req.SQL)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Query execution failed: %v", err), http.StatusBadRequest)
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get columns: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare result slice
	var results []interface{}

	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row into value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to scan row: %v", err), http.StatusInternalServerError)
			return
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte arrays to strings
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		s.sendError(w, fmt.Sprintf("Row iteration error: %v", err), http.StatusInternalServerError)
		return
	}

	// Send successful response
	response := QueryResponse{
		Success: true,
		Data:    results,
		Count:   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *QueryServer) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := QueryResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}