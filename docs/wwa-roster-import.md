# WWA roster chat → CAWForge

CAWForge does not directly access private ChatGPT conversations. Export the roster from the chat into a JSON file and save it as:

`%USERPROFILE%\\Documents\\CAWForge\\wwa_roster_import.json`

Example:

```json
{
  "wrestlers": [
    {
      "name": "Scarlett Raines",
      "picbase": "Maddy Westbeld",
      "alignment": "",
      "group": "Active Women"
    },
    {
      "name": "Karli Rae",
      "picbase": "",
      "alignment": "",
      "group": "Active Women"
    }
  ]
}
```

The example is illustrative, **not a complete roster export**. Supported fields are `name`, `picbase`, `alignment`, `group` and optional `mugshot` (ignored on new imports). The importer also accepts a top-level JSON array.

In CAWForge, open Roster Manager and click **Import WWA JSON**. The first click displays a preview count; the second click with the same file applies the merge. The importer rejects empty/duplicate names, matches existing wrestlers case-insensitively, never erases existing fields with blank incoming values, preserves existing mugshots, and backs up the local roster before writing.

The importer is **not a live chat sync**. It imports only the information actually included in the JSON file. Changes to the file between clicks require a new preview.
