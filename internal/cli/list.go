package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/matsubo/voice-memo-stt/internal/voicememos"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Voice Memos recordings",
	RunE: func(cmd *cobra.Command, args []string) error {
		recs, err := voicememos.Load(cmd.Context())
		if err != nil {
			return err
		}

		if listJSON {
			return json.NewEncoder(os.Stdout).Encode(recs)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TITLE\tDATE\tDURATION\tPATH")
		for _, r := range recs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				r.Title,
				r.Date.Format("2006-01-02 15:04"),
				r.DurationFormatted(),
				r.Path,
			)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
}
