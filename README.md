# Database From Scratch in Go

This repository follows the progression of building a small database from first principles in Go: file durability, indexing, B-tree storage, KV pages, free-list reuse, tables, range queries, secondary indexes, transactions, concurrency control, parsing, and a SQL-like query layer.

## What I Built

The project is intentionally organized by chapter so the implementation reads like a student working through the material step by step. Each chapter introduces the next concept without hiding the earlier design underneath a large "framework".

## Chapter Progress

- [x] Chapter 01 — From Files To Databases
- [x] Chapter 02 — Indexing Data Structures
- [x] Chapter 03 — B-Tree & Crash Recovery
- [x] Chapter 04 — B-Tree Node and Insertion
- [x] Chapter 05 — B-Tree Deletion and Testing
- [x] Chapter 06 — Append-Only KV Store
- [x] Chapter 07 — Free List: Recycle & Reuse
- [x] Chapter 08 — Tables on KV
- [x] Chapter 09 — Range Queries
- [x] Chapter 10 — Secondary Indexes
- [x] Chapter 11 — Atomic Transactions
- [x] Chapter 12 — Concurrency Control
- [x] Chapter 13 — SQL Parser
- [x] Chapter 14 — Query Language

## Architecture

SQL / Query Language
        ↓
Parser
        ↓
Query Executor
        ↓
Tables
        ↓
Secondary Indexes
        ↓
Transactions
        ↓
KV Store
        ↓
B+Tree
        ↓
Pages / Disk

## Running

```bash
go test ./...
go run ./cmd/db
```

Then open:

http://localhost:8080

The app serves a small browser UI that posts SQL-like statements to the built-in query executor. Tables and rows are currently held in memory and reset when the server restarts. Create tables with the columns you need; the executor does not assume a fixed table or column set:

```sql
CREATE TABLE products (sku TEXT, name TEXT, price DECIMAL(10,2))
INSERT INTO products (sku, name, price) VALUES ('P-100', 'Desk lamp', 19.50)
SELECT sku, name FROM products WHERE price >= 10
UPDATE products SET price = 21.00 WHERE sku = 'P-100'
ALTER TABLE products ADD COLUMN in_stock BOOLEAN
ALTER TABLE products RENAME COLUMN in_stock TO available
ALTER TABLE products DROP COLUMN available
DROP TABLE products
```

SQL keywords are case-insensitive. The supported DDL statements are `CREATE TABLE`, `DROP TABLE`, and `ALTER TABLE` with `ADD COLUMN`, `DROP COLUMN`, `RENAME COLUMN`, or `RENAME TO`. `SELECT` supports `*`, column projections, comparison predicates in `WHERE`, and multiple predicates joined with `AND`. `UPDATE` supports one or more column assignments and optional comparison predicates joined with `AND`; without `WHERE`, it updates every row. It returns the updated rows. Values can be quoted with single quotes; identifiers can be quoted with double quotes or backticks. `INSERT` uses a declared table schema, with an optional column list and a parenthesized `VALUES` list.
