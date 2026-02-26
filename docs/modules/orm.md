# orm Module

Optional ORM layer built on top of `db.Adapter`.

## Provides

- Generic table repository access (`Table("...")`)
- Query methods: `FindMany`, `FindFirst`, `FindFirstOrThrow`, `FindUnique`, `FindUniqueOrThrow`
- Write methods: `Create`, `Update`, `Upsert`, `Delete`
- Bulk methods: `CreateMany`, `CreateManyAndReturn`, `UpdateMany`, `UpdateManyAndReturn`, `DeleteMany`
- Raw SQL escape hatch via `Client.SQL()`
- Dialect-aware placeholders (`postgres`/`pg` vs `?`)
- Backward-compatible `Where` map (simple equality still works)

## Where Operators

- Logical: `AND`, `OR`, `NOT`
- Scalar: `equals`, `not`, `in`, `notIn`, `lt`, `lte`, `gt`, `gte`, `contains`, `startsWith`, `endsWith`, `isSet`, `isEmpty`
- Unsupported (returns `ErrUnsupportedOperator`): `some`, `every`, `none`, `has`, `hasEvery`, `hasSome` (these need dialect-specific composite/array semantics)

## Primary API

- `func New(adapter db.Adapter) *Client`
- `func NewWithConfig(adapter db.Adapter, cfg Config) *Client`
- `func (c *Client) SQL() db.Adapter`
- `func (c *Client) Table(name string) *Table`
- `func (t *Table) FindMany(ctx context.Context, opts FindOptions) ([]Record, error)`
- `func (t *Table) FindFirst(ctx context.Context, opts FindOptions) (Record, error)`
- `func (t *Table) FindFirstOrThrow(ctx context.Context, opts FindOptions) (Record, error)`
- `func (t *Table) FindUnique(ctx context.Context, where Where, columns ...string) (Record, error)`
- `func (t *Table) FindUniqueOrThrow(ctx context.Context, where Where, columns ...string) (Record, error)`
- `func (t *Table) Create(ctx context.Context, values Values) (db.Result, error)`
- `func (t *Table) Update(ctx context.Context, where Where, patch Values) (int64, error)`
- `func (t *Table) Upsert(ctx context.Context, where Where, create Values, update Values) (Record, error)`
- `func (t *Table) Delete(ctx context.Context, where Where) (int64, error)`
- `func (t *Table) CreateMany(ctx context.Context, rows []Values) (int64, error)`
- `func (t *Table) CreateManyAndReturn(ctx context.Context, rows []Values, columns []string) ([]Record, error)`
- `func (t *Table) UpdateMany(ctx context.Context, where Where, patch Values) (int64, error)`
- `func (t *Table) UpdateManyAndReturn(ctx context.Context, where Where, patch Values, columns []string) ([]Record, error)`
- `func (t *Table) DeleteMany(ctx context.Context, where Where) (int64, error)`
