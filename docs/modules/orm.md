# orm Module

Optional ORM layer built on top of `db.Adapter`.

## Provides

- Generic table repository access (`Table("...")`)
- CRUD operations (`Create`, `Update`, `Delete`)
- Record queries (`FindOne`, `FindMany`)
- Raw SQL escape hatch via `Client.SQL()`
- Dialect-aware placeholders (`postgres`/`pg` vs `?`)

## Primary API

- `func New(adapter db.Adapter) *Client`
- `func NewWithConfig(adapter db.Adapter, cfg Config) *Client`
- `func (c *Client) SQL() db.Adapter`
- `func (c *Client) Table(name string) *Table`
- `func (t *Table) Create(ctx context.Context, values Values) (db.Result, error)`
- `func (t *Table) FindOne(ctx context.Context, opts FindOptions) (Record, error)`
- `func (t *Table) FindMany(ctx context.Context, opts FindOptions) ([]Record, error)`
- `func (t *Table) Update(ctx context.Context, where Where, patch Values) (int64, error)`
- `func (t *Table) Delete(ctx context.Context, where Where) (int64, error)`
