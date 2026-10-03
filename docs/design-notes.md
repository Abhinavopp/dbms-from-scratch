# Design Notes

## Chapter 01
What problem was discovered?

This chapter focuses on the simplest durable primitive: writing bytes to disk in a way that is safe across crashes. The main issue is not file I/O itself, but making sure a write is either complete or not visible.

## Chapter 02
Why isn't a simple sorted array enough?

A sorted array supports fast lookup when the data is tiny, but once the dataset grows, inserts become expensive and the structure ends up shifting large portions of memory. The chapter introduces the idea of an index as a stand-in for a more scalable search structure.

## Chapter 03
Why does a disk-based B+Tree need crash-safe updates?

A B+Tree on disk cannot just mutate pointers in place without considering the fact that an update may be interrupted halfway through. The design needs a consistent page structure and careful update ordering so the tree can be recovered from stable pages.

## Chapter 04
What changes during insertion?

Insertion is where the tree begins to split pages and maintain key order. The important property is that the structure stays balanced and that keys remain sorted across the tree.

## Chapter 05
Why is deletion more subtle than insertion?

Deletion can create underflow and forces rebalancing across siblings. If the tree does not merge or redistribute keys correctly, searches will become inconsistent and the B+Tree order invariant will break.

## Chapter 06
What does the KV layer add?

The key-value store isolates the page layout and lets higher layers think in terms of set/get/delete operations rather than raw disk bytes. This is the first point where storage becomes a reusable abstraction.

## Chapter 07
Why add a free list?

The free list reuses page numbers instead of endlessly allocating new files or extending storage without plan. That design is important for long-running systems where old pages are deleted or overwritten.

## Chapter 08
What is the role of tables?

A table maps rows and schema into the KV model. This adds a data model above raw storage and starts to resemble a real relational table layer.

## Chapter 09
Why do range queries matter?

Range reads depend on ordered traversal. Once keys are stored in sorted order, a query can move iteratively through the relevant pages without scanning the entire dataset.

## Chapter 10
Why are secondary indexes useful?

A primary index is usually the record's key, but many queries filter on other fields. Secondary indexes allow those lookups without scanning every row, at the cost of additional maintenance.

## Chapter 11
What is the transaction boundary?

Transactions group a set of writes so they appear atomic. The goal is that either the whole write set is durable or nothing is visible after a failure.

## Chapter 12
Why is concurrency control different from atomicity?

Even with correct transactional semantics, concurrent readers and writers can interleave and produce incorrect results. A coordination layer is needed to preserve visibility and consistency under concurrency.

## Chapter 13
Why parse SQL before executing it?

A query parser turns strings into a structured representation. That makes validation, optimization, and execution much easier than operating on raw text in the executor.

## Chapter 14
What does the executor connect together?

The final executor turns parsed statements into actual reads and writes on the table and index layer. This is the point where the database stops being just a storage engine and becomes a query engine.
