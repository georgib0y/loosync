package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSqliteMemoryOpens(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")

	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("create table foo (id integer not null primary key, name text);")

	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("insert into foo(id, name) values(1, \"george\")")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("select id, name from foo")
	if err != nil {
		t.Fatal(err)
	}

	if !rows.Next() {
		t.Fatal("Did not return any results")
	}

	var id int
	var name string

	if err = rows.Scan(&id, &name); err != nil {
		t.Fatal(err)
	}

	if id != 1 || name != "george" {
		t.Fatalf("id and name is %d, %s but expected 1, george", id, name)
	}

	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
}
