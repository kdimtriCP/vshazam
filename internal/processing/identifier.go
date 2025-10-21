package processing

import "context"

type identifier struct{}

type Identifier interface {
	Run(ctx context.Context, process func() error) error
}
