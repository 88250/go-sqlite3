//go:build cgo

package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestQueryMultipleStatementsTrailingComments(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, tail := range []string{"", ";", ";; -- SELECT 2", "; /* ; DELETE FROM items */ ; -- tail\n", "; /* unterminated"} {
		var got string
		if err = db.QueryRow("SELECT ?"+tail, "quoted; value").Scan(&got); err != nil || got != "quoted; value" {
			t.Fatalf("tail=%q: got=%q, err=%v", tail, got, err)
		}
	}
}

func TestQueryMultipleStatementsHonorsCancellation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		WITH RECURSIVE sequence(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM sequence WHERE n < 1000000000)
		SELECT sum(n) FROM sequence;
		SELECT 1;
	`)
	if rows != nil {
		rows.Close()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("intermediate query did not honor cancellation: %v", err)
	}
}

func TestQueryMultipleStatementsExecuteInOrder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		CREATE TABLE items (id INTEGER PRIMARY KEY, n INTEGER NOT NULL);
		INSERT INTO items VALUES (?, ?);
		UPDATE items SET n = n + 1 RETURNING n;
		SELECT n FROM items;
	`, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if !rows.Next() {
		t.Fatalf("missing final result: %v", rows.Err())
	}
	if err = rows.Scan(&n); err != nil || n != 11 {
		t.Fatalf("final result = %d, %v", n, err)
	}
	if rows.Next() || rows.Err() != nil {
		t.Fatalf("unexpected final results: %v", rows.Err())
	}
	if err = rows.Close(); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRowContext(ctx, "SELECT n FROM items").Scan(&n); err != nil || n != 11 {
		t.Fatalf("write was skipped or repeated: %d, %v", n, err)
	}
}

func TestQueryMultipleStatementsStopsAtExecutionError(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err = db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY); INSERT INTO items VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		"INSERT INTO items VALUES (1); DELETE FROM items",
		"INSERT INTO items VALUES (1) RETURNING id; DELETE FROM items",
		"SELECT abs(-9223372036854775808); DELETE FROM items",
	} {
		rows, queryErr := db.Query(stmt)
		if rows != nil {
			rows.Close()
		}
		if queryErr == nil {
			t.Fatalf("intermediate execution failure was ignored: %s", stmt)
		}
		var count int
		if err = db.QueryRow("SELECT count(*) FROM items").Scan(&count); err != nil || count != 1 {
			t.Fatalf("statements after failure ran: count=%d, %v", count, err)
		}
	}
}
