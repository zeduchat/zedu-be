# Engineering Agent Guidelines & Coding Standards

This document specifies the mandatory engineering standards, architectural patterns, and code style conventions for contributing to the `telex_be` codebase.

---

## 1. Architectural Layout & Layering

The codebase follows a structured layered architecture:

- **`pkg/controller/<feature>/`**: HTTP request handlers and Gin controller bindings.
- **`services/<feature>/`**: Business logic orchestration layer.
- **`internal/models/`**: Domain models, database structs, request/response DTOs, and schema mappings.
- **`pkg/repository/storage/`**: Database and storage engine connections & helper wrappers:
  - `postgresql/`: PostgreSQL GORM database helper wrappers (`CheckExists`, `SelectOneFromDb`, `CreateOneRecord`, `UpdateFields`, `DeleteRecordFromDb`).
  - `elastic/`: Elasticsearch document operations (`SelectAll`, `AddDocument`, `UpdateDocument`, `DeleteDocument`, `DeleteByQuery`, `SelectByID`).
  - `redis/`, `minio/`, `typesense/`, `mongodb/`: Specialized storage adapters.
- **`internal/config/`**: Environment variable loading, Viper configurations, and application setting structs.
- **`utility/`**: Logger, HTTP response builders (`BuildSuccessResponse`, `BuildErrorResponse`), and generic utility functions.

---

## 2. Function Parameter Discipline (Max 3 Parameter Rule)

To maintain clean function signatures and readability:

1. **Maximum Positional Parameters**:
   - Functions and methods SHOULD NOT accept more than **3 positional parameters**.
   
2. **Encapsulation in Structs**:
   - If a function requires 4 or more arguments, group them into a dedicated parameter or options struct (such as `models.IDS`, `models.SearchQueryFiltersKeywords`, or feature-specific Request DTOs).

---

## 3. HTTP Controller & Handler Conventions

1. **Controller Composition**:
   - Controllers are defined as structs embedding core dependencies (database connection, struct validator, logger instance, and external request handler).

2. **Request Parsing & Validation**:
   - Bind incoming JSON payloads using the standard context JSON binder (`c.ShouldBindJSON`).
   - Validate struct constraints using the struct validator (`base.Validator.Struct`) with struct tags (`binding:"required"`, `validate:"required"`).

3. **Standardized HTTP Responses**:
   - Always format API responses using the standard utility response builders (`utility.BuildSuccessResponse` or `utility.BuildErrorResponse`) and standard HTTP status codes.

4. **Claims Extraction**:
   - Extract JWT claims via middleware context (`userClaims`) or authorization middleware helpers (`middleware.GetUserClaims`).

---

## 4. Database & Storage Layer Wrappers

1. **PostgreSQL Wrappers**:
   - Use established helper abstractions in `pkg/repository/storage/postgresql` such as `CheckExists`, `SelectOneFromDb`, `CreateOneRecord`, `UpdateFields`, and `DeleteRecordFromDb`.

2. **Elasticsearch Wrappers**:
   - Utilize helper functions in `pkg/repository/storage/elastic`.
   - Index names must reference `models.ThreadIndexName` and `models.MessageIndexName` (dynamically prefixed based on `ELASTIC_INDEX_PREFIX`).

---

## 5. Centralized Testing Pattern

1. **Test File Directory**:
   - All unit and integration test files MUST reside under the central `tests/` directory following the `tests/test_<feature>/` pattern.

2. **Package Naming**:
   - Test files inside `tests/test_<feature>/` must use package `test_<feature>` (e.g. `package test_config`, `package test_message`).

3. **No Scattered Tests**:
   - Do NOT place unit test files directly inside `internal/` or `pkg/` directories.

---

## 6. Comment & Style Discipline

1. **Omit Redundant Comments**:
   - Do not add comments that merely restate what the code clearly expresses.

2. **Strict Ban on Numbered Comments**:
   - **NEVER use numbered comments** (e.g. `// 1. Parse payload`, `// 2. Validate user`).

3. **Non-Obvious Rationale Only**:
   - Use comments only when documenting non-obvious business rules, edge cases, workarounds, or limitations.

4. **Clean Commits**:
   - Remove temporary `TODO` or `FIXME` comments before committing code.

---

## 7. Security, Input Validation & Environment Hygiene

1. **Input Sanitization**:
   - Validate and sanitize all external user inputs before processing.

2. **No SQL / Shell Concatenation**:
   - Always use GORM parameterized clauses or ORM query builders. Never concatenate raw SQL strings.

3. **Environment Hygiene**:
   - Never hardcode secrets, ports, URLs, or API keys in source files.
   - Register new environment parameters in `BaseConfig` (`internal/config/env.go`) and document them in `app-sample.env`.

---

## 8. Workflow Directive

- **Proposal Mandatory**: DO NOT start any task without writing an implementation proposal and obtaining user approval first.

---

## 9. Commit Message Format

- **Format Requirement**: All commit messages MUST follow the format `action: what was done` (e.g. `add: elastic index prefix`, `add: codebase engineering standards`).
- **Conciseness**: Keep commit titles concise and descriptive (no more than 4 words).

