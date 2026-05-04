# Promise.all that quietly drops failures

The on-call engineer reports that the **bulk upload importer** silently
"succeeds" even when several individual uploads fail and rows are missing
downstream. They suspect the failure handling, not the uploads themselves.

Read `bug/notes.md` and `bug/uploader.ts`, then choose the option that best
explains the bug.
