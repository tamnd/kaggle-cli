package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/kaggle-cli/kaggle"
)

func (a *App) datasetsCmd() *cobra.Command {
	var (
		search string
		sortBy string
		page   int
		tagIDs string
	)

	cmd := &cobra.Command{
		Use:   "datasets",
		Short: "List and search Kaggle datasets",
		Long: `List and search public Kaggle datasets.

The Kaggle datasets API is open without authentication.

Sort options: hottest, votes, updated, active, published

Examples:
  kaggle datasets                           # hottest datasets
  kaggle datasets --search titanic          # search by keyword
  kaggle datasets --sort votes -n 50        # top 50 by votes
  kaggle datasets --search nlp --sort updated -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			if search != "" {
				a.progressf("searching datasets for %q...", search)
			} else {
				a.progressf("fetching %s datasets...", sortBy)
			}
			datasets, err := a.client.Datasets(cmd.Context(), kaggle.DatasetOptions{
				Search: search,
				SortBy: sortBy,
				Page:   page,
				Limit:  n,
				TagIDs: tagIDs,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(datasets, len(datasets))
		},
	}

	cmd.Flags().StringVarP(&search, "search", "s", "", "keyword to search for")
	cmd.Flags().StringVar(&sortBy, "sort", "hottest", "sort order: hottest|votes|updated|active|published")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based)")
	cmd.Flags().StringVar(&tagIDs, "tags", "", "comma-separated tag IDs to filter by")

	return cmd
}
