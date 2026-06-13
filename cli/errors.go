package cli

import (
	"errors"

	"github.com/tamnd/kaggle-cli/kaggle"
)

func isNotFound(err error) bool {
	return errors.Is(err, kaggle.ErrNotFound)
}
