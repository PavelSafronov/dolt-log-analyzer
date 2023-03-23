# Dolt Log Analyzer

This is a simple tool for analyzing dolt query logs.
The tool parses the logs, pulls out the queries, parses them into a tree, and then prints out the tree for each query.
The results are written out to a file and optionally to the console.

## Installing from source

1. Clone the repo
2. Run `make` at the root of the repo to build, test, and install the tool

## Usage

Create or download a dolt query log file `log.txt` that's of this format:

```text
* Starting pprof server on port 6060.
* Go to http://localhost:6060/debug/pprof in a browser to see supported endpoints.
*
* Known endpoints are:
*   /allocs: A sampling of all past memory allocations
*   /block: Stack traces that led to blocking on synchronization primitives
*   /cmdline: The command line invocation of the current program
*   /goroutine: Stack traces of all current goroutines
*   /heap: A sampling of memory allocations of live objects. You can specify the gc GET parameter to run GC before taking the heap sample.
*   /mutex: Stack traces of holders of contended mutexes
*   /profile: CPU profile. You can specify the duration in the seconds GET parameter. After you get the profile file, use the go tool pprof command to investigate the profile.
*   /threadcreate: Stack traces that led to the creation of new OS threads
*   /trace: A trace of execution of the current program. You can specify the duration in the seconds GET parameter. After you get the trace file, use the go tool trace command to investigate the trace.

Starting server with Config HP="0.0.0.0:3306"|T="28800000"|R="false"|L="debug"
2023-03-22T21:54:43Z INFO [conn 1] NewConnection {DisableClientMultiStatements=false}
2023-03-22T21:54:43Z DEBUG [conn 1] Starting query {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIE5BTUVTIHV0ZjhtYjQ=}
2023-03-22T21:54:43Z DEBUG [conn 1] Query finished in 25 ms {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIE5BTUVTIHV0ZjhtYjQ=}
2023-03-22T21:54:43Z DEBUG [conn 1] Starting query {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIGF1dG9jb21taXQ9MA==}
2023-03-22T21:54:43Z DEBUG [conn 1] Query finished in 1 ms {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIGF1dG9jb21taXQ9MA==}
2023-03-22T21:54:43Z DEBUG [conn 1] Starting query {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIGF1dG9jb21taXQ9MQ==}
2023-03-22T21:54:43Z DEBUG [conn 1] Query finished in 1 ms {connectTime=2023-03-22T21:54:43Z, connectionDb=nautobot, query=U0VUIGF1dG9jb21taXQ9MQ==}
```

Run the tool:

```bash
dolt-log-analyzer -log log.txt -out output.txt
```

The result will look like this:

```text
line 18, query tree: 
SET character_set_client = utf8mb4 (longtext), SET character_set_connection = utf8mb4 (longtext), SET character_set_results = utf8mb4 (longtext)

--------------------------------------------------
line 20, query tree: 
SET autocommit = 0 (tinyint)

--------------------------------------------------
line 22, query tree: 
SET autocommit = 1 (tinyint)

--------------------------------------------------
```