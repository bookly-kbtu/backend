package domain

import "context"

// TxManager runs fn in one DB transaction. Repositories called with the
// ctx passed to fn take part in that transaction.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
