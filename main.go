package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var version = "dev" // GoReleaser will overwrite "dev" with the tag

func main() {
	var rootCmd = &cobra.Command{
		Use:     "cnote",
		Short:   "cnote: A casual, ephemeral note-taking tool",
		Long:    `cnote is an in-memory note tool. Notes persist only while the list is not empty.`,
		Version: version,
	}

	// --- HIDDEN DAEMON COMMAND ---
	// This is not meant to be run by humans. It is spawned by the client.
	var daemonCmd = &cobra.Command{
		Use:    "daemon",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			StartDaemon()
		},
	}

	// --- ADD ---
	var addCmd = &cobra.Command{
		Use:   "add [note text]",
		Short: "Add a note (Starts session if empty)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(true)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			defer client.Close()

			pinFlag, err := cmd.Flags().GetBool("pin")
			if err != nil {
				fmt.Println("Error retrieving pin flag:", err)
				return
			}

			var reply NoteReply
			err = client.Call("NoteService.Add", AddArgs{
				Text:   args[0],
				Pinned: pinFlag,
			}, &reply)

			if err != nil {
				fmt.Println("RPC Error:", err)
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- LIST ---
	var listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all notes",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false) // false = do not start daemon if missing
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()

			var reply ListReply
			err = client.Call("NoteService.List", EmptyArgs{}, &reply)
			if err != nil {
				fmt.Println("RPC Error:", err)
				return
			}

			if len(reply.Notes) == 0 {
				fmt.Println("No notes found.")
				return
			}

			// Tabwriter for clean columns
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\t \tTIME\tNOTE")
			fmt.Fprintln(w, "--\t-\t----\t----")
			for _, n := range reply.Notes {
				pinMarker := ""
				if n.Pinned {
					pinMarker = "📌"
				}
				dateStr := n.CreatedAt.Format("15:04")
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", n.ID, pinMarker, dateStr, n.Text)
			}
			w.Flush()
		},
	}

	// --- REMOVE ---
	var removeCmd = &cobra.Command{
		Use:     "remove [id]",
		Aliases: []string{"rm"},
		Short:   "Remove a note ('first', 'last', or ID)",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()

			var reply NoteReply
			err = client.Call("NoteService.Remove", IDArgs{IDStr: args[0]}, &reply)
			if err != nil {
				fmt.Println("Error:", err) // Likely "ID not found"
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- CLEAR ---
	var clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "Clear all notes and stop session",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()

			var reply NoteReply
			client.Call("NoteService.Clear", EmptyArgs{}, &reply)
			fmt.Println(reply.Message)
		},
	}

	// --- PIN/UNPIN/SHOW Wrappers ---
	// Helper to reduce code duplication for simple ID commands
	runIDCommand := func(method string, id string) {
		client, err := getClient(false)
		if err != nil {
			fmt.Println("No active session.")
			return
		}
		defer client.Close()
		var reply NoteReply
		if err := client.Call(method, IDArgs{IDStr: id}, &reply); err != nil {
			fmt.Println("Error:", err)
			return
		}

		if method == "NoteService.Show" {
			n := reply.Note
			fmt.Printf("--- Note %d ---\n", n.ID)
			fmt.Printf("Pinned:  %v\n", n.Pinned)
			fmt.Printf("Created: %s\n", n.CreatedAt.Format(time.Kitchen))
			fmt.Printf("Content: %s\n", n.Text)
		} else {
			fmt.Println(reply.Message)
		}
	}

	var pinCmd = &cobra.Command{
		Use: "pin [id]", Short: "Pin a note", Args: cobra.ExactArgs(1),
		Run: func(c *cobra.Command, a []string) { runIDCommand("NoteService.Pin", a[0]) },
	}

	var unpinCmd = &cobra.Command{
		Use: "unpin [id]", Short: "Unpin a note", Args: cobra.ExactArgs(1),
		Run: func(c *cobra.Command, a []string) { runIDCommand("NoteService.Unpin", a[0]) },
	}

	var showCmd = &cobra.Command{
		Use: "show [id]", Short: "Show full details", Args: cobra.ExactArgs(1),
		Run: func(c *cobra.Command, a []string) { runIDCommand("NoteService.Show", a[0]) },
	}

	// Register flag before Execute
	addCmd.Flags().BoolP("pin", "p", false, "Pin the note immediately")

	// Add all commands to rootCmd
	rootCmd.AddCommand(daemonCmd, addCmd, listCmd, removeCmd, clearCmd, pinCmd, unpinCmd, showCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
