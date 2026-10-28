// Package queries is Turnia's SQL as Go functions. The SQL is written by hand
// in the .sql files here; sqlc reads them, checks them against the schema in
// the migrations, and writes the .sql.go files, db.go and models.go. Don't
// edit those: change the SQL and run `make generate`.
//
// queries.New takes a pool, a single connection or a transaction, so the same
// query can run inside or outside one.
package queries
