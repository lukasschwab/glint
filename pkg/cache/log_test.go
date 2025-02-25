package cache_test

import (
	"testing"

	"github.com/lukasschwab/glint/pkg/cache"
)

func TestLog(t *testing.T) {
	var _ cache.Logger = cache.TestLogger{t}
	var _ cache.Logger = cache.NoopLogger{}
}
