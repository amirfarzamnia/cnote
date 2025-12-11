package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "tnote"}

	// --- Internal Command: Daemon (Hidden) ---
	var daemonCmd = &cobra.Command{
		Use:    "daemon",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			StartDaemon()
		},
	}

	// --- ADD ---
	var addCmd = &cobra.Command{
		Use:   "add [note]",
		Short: "Add a note to memory",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			text := args[0]
			// True = Auto-start daemon if missing
			client, err := getClient(true)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			defer client.Close()

			var reply NoteReply
			err = client.Call("NoteService.Add", AddArgs{Text: text}, &reply)
			if err != nil {
				fmt.Println("RPC Error:", err)
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- LIST ---
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all notes",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active notes in memory.")
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

			// Pretty print with TabWriter
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tP\tCREATED\tNOTE")
			for _, n := range reply.Notes {
				pinMarker := ""
				if n.Pinned {
					pinMarker = "*"
				}
				// Format date human-readable
				dateStr := n.CreatedAt.Format("15:04:05")
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", n.ID, pinMarker, dateStr, n.Text)
			}
			w.Flush()
		},
	}

	// --- REMOVE ---
	var removeCmd = &cobra.Command{
		Use:   "remove [id]",
		Short: "Remove a note (supports 'first', 'last')",
		Args:  cobra.ExactArgs(1),
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
				fmt.Println("Error:", err)
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- CLEAR ---
	var clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "Remove all notes and stop session",
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

	// --- PIN ---
	var pinCmd = &cobra.Command{
		Use:   "pin [id]",
		Short: "Pin a note",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()
			var reply NoteReply
			if err := client.Call("NoteService.Pin", IDArgs{IDStr: args[0]}, &reply); err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- UNPIN ---
	var unpinCmd = &cobra.Command{
		Use:   "unpin [id]",
		Short: "Unpin a note",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()
			var reply NoteReply
			if err := client.Call("NoteService.Unpin", IDArgs{IDStr: args[0]}, &reply); err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println(reply.Message)
		},
	}

	// --- SHOW ---
	var showCmd = &cobra.Command{
		Use:   "show [id]",
		Short: "Show details of a specific note",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := getClient(false)
			if err != nil {
				fmt.Println("No active session.")
				return
			}
			defer client.Close()
			var reply NoteReply
			if err := client.Call("NoteService.Show", IDArgs{IDStr: args[0]}, &reply); err != nil {
				fmt.Println("Error:", err)
				return
			}
			n := reply.Note
			fmt.Printf("ID:      %d\n", n.ID)
			fmt.Printf("Pinned:  %v\n", n.Pinned)
			fmt.Printf("Created: %s\n", n.CreatedAt.Format(time.RFC1123))
			fmt.Printf("Text:    %s\n", n.Text)
		},
	}

	rootCmd.AddCommand(daemonCmd, addCmd, listCmd, removeCmd, clearCmd, pinCmd, unpinCmd, showCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
