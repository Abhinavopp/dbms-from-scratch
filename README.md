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
DELETE FROM products WHERE sku = 'P-100'
ALTER TABLE products ADD COLUMN in_stock BOOLEAN
ALTER TABLE products RENAME COLUMN in_stock TO available
ALTER TABLE products DROP COLUMN available
DROP TABLE products
```

SQL keywords are case-insensitive. `CREATE TABLE` accepts column definitions, a single-column `PRIMARY KEY`, and single-column `INDEX` definitions. `INSERT` accepts an optional column list and one or more parenthesized `VALUES` rows. `SELECT` supports column projections, arithmetic/logical expressions with `AS` aliases, `INDEX BY`, `FILTER`, `WHERE`, and `LIMIT`; `LIMIT offset, count` is also accepted. `INDEX BY` must refer to a declared index or primary-key column and supports equality and comparison ranges, including reversed comparisons such as `20 < age`. `FILTER` and `WHERE` accept boolean expressions with `AND`, `OR`, and `NOT`. Expressions support comparisons, arithmetic, unary minus, integer/decimal literals, strings, booleans, and column references. `UPDATE` supports multiple expression assignments, optionally filtered by `WHERE`; `DELETE FROM` supports an optional `WHERE` expression. Non-column expressions in a `SELECT` projection require an `AS` alias.

This query executor is an educational in-memory implementation: rows and indexes are not persisted and are reset when the server restarts. Its index structures are maintained by the executor and used for `INDEX BY` scans; they are separate from the earlier chapter demonstrations of storage structures.
