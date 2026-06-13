package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/kaggle-cli/kaggle"
)

func (a *App) competitionsCmd() *cobra.Command {
	var (
		search   string
		sortBy   string
		category string
		page     int
	)

	cmd := &cobra.Command{
		Use:   "competitions",
		Short: "List Kaggle competitions",
		Long: `List Kaggle competitions.

This endpoint requires Kaggle authentication. Set KAGGLE_USERNAME and
KAGGLE_KEY environment variables (get your key from
https://www.kaggle.com/settings under "API").

Category options: all, featured, research, recruitment, gettingStarted,
  masters, playground

Sort options: latestDeadline, earliestDeadline, numberOfTeams, recentlyCreated

Examples:
  kaggle competitions                             # competitions by latest deadline
  kaggle competitions --search vision             # search by keyword
  kaggle competitions --category featured -n 10   # top featured competitions
  kaggle competitions --sort numberOfTeams -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			if search != "" {
				a.progressf("searching competitions for %q...", search)
			} else {
				a.progressf("fetching %s competitions...", category)
			}
			comps, err := a.client.Competitions(cmd.Context(), kaggle.CompetitionOptions{
				Search:   search,
				Category: category,
				SortBy:   sortBy,
				Page:     page,
				Limit:    n,
			})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(comps, len(comps))
		},
	}

	cmd.Flags().StringVarP(&search, "search", "s", "", "keyword to search for")
	cmd.Flags().StringVar(&sortBy, "sort", "latestDeadline", "sort order: latestDeadline|earliestDeadline|numberOfTeams|recentlyCreated")
	cmd.Flags().StringVar(&category, "category", "all", "category: all|featured|research|recruitment|gettingStarted|masters|playground")
	cmd.Flags().IntVar(&page, "page", 1, "page number (1-based)")

	return cmd
}
