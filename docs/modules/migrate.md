# migrate Module

SQL migration loading and execution.

## Provides

- Migration file loader from directory
- Engine operations: `Up`, `Down`, `Status`
- Version tracking table management
- Optional dialect-specific migration file selection

## Supported file naming

- `0001_init.up.sql`
- `0001_init.down.sql`
- `0001_init.pg.up.sql` (dialect-specific)

## Primary API

- `func Load(dir, dialect string) ([]Migration, error)`
- `func NewEngine(adapter db.Adapter, dialect string) *Engine`
- `func (e *Engine) Up(ctx context.Context, migrations []Migration, steps int) (int, error)`
- `func (e *Engine) Down(ctx context.Context, migrations []Migration, steps int) (int, error)`
- `func (e *Engine) Status(ctx context.Context, migrations []Migration) ([]Status, error)`
