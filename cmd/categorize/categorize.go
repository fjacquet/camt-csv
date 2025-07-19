// Package categorize handles transaction categorization commands
package categorize

import (
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/config"

	"github.com/spf13/cobra"
)

// Cmd represents the categorize command
var Cmd = &cobra.Command{
	Use:   "categorize",
	Short: "Categorize transactions using Gemini model",
	Long:  `Categorize transactions based on the party's name and typical activity using Gemini model.`,
	Run:   categorizeFunc,
}

func init() {
	// Category command flags
	Cmd.Flags().StringVarP(&root.PartyName, "party", "p", "", "Party name to categorize")
	Cmd.Flags().BoolVarP(&root.IsDebtor, "debtor", "d", false, "Whether the party is a debtor (default: creditor)")
	Cmd.Flags().StringVarP(&root.Amount, "amount", "a", "", "Transaction amount (optional)")
	Cmd.Flags().StringVarP(&root.Date, "date", "t", "", "Transaction date (optional)")
	Cmd.Flags().StringVarP(&root.Info, "info", "n", "", "Additional transaction info (optional)")
	Cmd.MarkFlagRequired("party")
}

func categorizeFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Categorize command called")

	// Ensure the environment variables are loaded
	config.LoadEnv()

	if root.PartyName != "" {
		// Create a transaction object to categorize
		transaction := categorizer.Transaction{
			PartyName: root.PartyName,
			IsDebtor:  root.IsDebtor,
			Amount:    root.Amount,
			Date:      root.Date,
			Info:      root.Info,
		}

		// Categorize the transaction
		category, err := categorizer.CategorizeTransaction(transaction)
		if err != nil {
			root.Log.Errorf("Error categorizing transaction: %v", err)
		} else {
			root.Log.Infof("Category: %s", category.Name)

			// The mappings are automatically updated through CategorizeTransaction
			root.Log.Infof("Transaction categorized as: %s", category.Name)
		}
	} else {
		root.Log.Error("Party name is required for categorization")
	}
}
