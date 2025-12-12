# cnote 🎩

**Casual Note** is a minimalist, ephemeral CLI note-taking tool for Linux/macOS.

It is designed for the "scratchpad" workflow: you need to remember something _right now_, but you don't need it forever.

> **The Hook:** `cnote` has no database and no config file. Notes live in RAM.
> When you add a note, a lightweight session starts.
> When you clear your list (or remove the last note), the session dies and frees all memory.

## 🚀 Installation

```bash
git clone [https://github.com/yourusername/cnote.git](https://github.com/yourusername/cnote.git)
cd cnote
go build -o cnote .
sudo mv cnote /usr/local/bin/
```

## 🎩 Usage

**1. Start a session**
Just add a note. If `cnote` isn't running, it starts itself.

```bash
cnote add "Deploy to production at 4pm"
# 🎩 Note added (ID: 1)
```

**2. Add more**

```bash
cnote add Buy milk
cnote add "Check server logs"
```

**3. View notes**

```bash
cnote list
# ID   	 	TIME	NOTE
# --   	-	----	----
# 1    		15:30	Deploy to production at 4pm
# 2    		15:31	Buy milk
# 3    		15:32	Check server logs
```

**4. Pin important stuff**

```bash
cnote pin 1
# 📌 Pinned note 1
```

**5. Smart Removal**
You can use IDs, or keywords `first` and `last`.

```bash
cnote remove last
# 🗑️ Removed note 3
```

**6. The "Done" Button**
When you clear the list, `cnote` shuts down completely.

```bash
cnote clear
# ✨ All notes cleared. Session ended.
```

## 🧠 Under the Hood (Architecture)

`cnote` is built for maximum efficiency using a **Client-Daemon** architecture hidden inside a single binary.

1.  **Lazy Loading:** When you run `cnote add`, the client checks for a Unix Domain Socket (`/tmp/cnote.sock`). If missing, it silently spawns a background process (the daemon).
2.  **In-Memory:** The daemon holds your notes in a Go slice (RAM). No disk I/O, no JSON files, no SQLite.
3.  **Aggressive Garbage Collection:** Every time a note is removed, the daemon checks the list size. If `count == 0`, the daemon calls `os.Exit(0)`, instantly returning all resources to the OS.

This ensures you never have a stray background service eating RAM when you aren't actually working on something.

## 📜 License

MIT
