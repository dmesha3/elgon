# migrate Module

SQL migration loading and execution.

## Provides

- Migration file loader from directory
- Engine operations: `Up`, `Down`, `Status`
- Model-to-SQL file generation (`generate`) with random migration suffixes
- Version tracking table management
- Optional dialect-specific migration file selection

## Supported file naming

- `0001_init.up.sql`
- `0001_init.down.sql`
- `0001_init.pg.up.sql` (dialect-specific)
- `20260228094510_init_ab12cd.up.sql` (generated)

## CLI Generate

Generate migration files from exported Go model structs:

```bash
elgon migrate generate \
  -models "internal/domain/*.go" \
  -dir migrations \
  -dialect sqlite \
  -name init \
  -apply
```

Flags:

- `-models`: comma-separated file paths or globs (required for `generate`)
- `-name`: migration base name (default `autogen`)
- `-apply`: apply the generated migration immediately
- `-driver`, `-dsn`: DB settings used only when `-apply` is set

## Primary API

- `func Load(dir, dialect string) ([]Migration, error)`
- `func GenerateFromModelFiles(dir, dialect, name string, modelFiles []string) (GeneratedMigration, error)`
- `func NewEngine(adapter db.Adapter, dialect string) *Engine`
- `func (e *Engine) Up(ctx context.Context, migrations []Migration, steps int) (int, error)`
- `func (e *Engine) Down(ctx context.Context, migrations []Migration, steps int) (int, error)`
- `func (e *Engine) Status(ctx context.Context, migrations []Migration) ([]Status, error)`
