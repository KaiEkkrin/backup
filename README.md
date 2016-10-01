## Example

backup.json
```
{
  "BaseName":   "mybackup",
  "Path":       "/",
  "Excludes":   ["/dev", "/proc", "/sys"],
  "Passphrase": "keepmesecret0"
}
```

### To create a backup

```
backup -job /path/to/backup.json -backup
```

This creates the files `mybackup_seen.db.kblob` and `mybackup_<datetime>.tar.kblob` in `/path/to/`.  If they already exist, it updates `mybackup_seen.db.kblob` and creates a new `mybackup_<current_datetime>.tar.kblob` with the incremental changes.

To secure your backup, save the `kblob` files to offline storage and the `json` file somewhere else, e.g. in your password safe.

### To verify and restore your backup

```
backup -job /path/to/backup.json -test
```

This verifies the integrity of the `kblob` files and prints out the list of files that have been backed up.

```
backup -job /path/to/backup.json -restore
```

This restores files out of the backup.

### The -prefix option

If you use a snapshotting filesystem, do this to backup your snapshot:

```
backup -job /path/to/backup.json -backup -prefix /path/to/current-snapshot
```

The backup files will not include `/path/to/current-snapshot` in the path and will be deduplicated with files backed up from `/path/to/an-earlier-snapshot` if they have not changed.

Do this to restore your backup into `/path/to/my-old-system` so that you can inspect and cherry-pick the contents:

```
backup -job /path/to/backup.json -restore -prefix /path/to/my-old-system
```

## About backup

It is a file archiving system for Windows and Linux platforms.  It uses `tar` as a file container and [komblobulate](https://github.com/kaiekkrin/komblobulate) to encrypt and add error resistance to the files.  You can unpack the backup files yourself using [kblob_cmd](https://github.com/kaiekkrin/kblob_cmd) to recover the tar file within (which is gzip'd).

Backup saves file mtime, uid, gid and permissions on Linux, but does not support extended attributes.  On Windows systems, it does not support file ACLs.

Backup supports multiple jobs in one go -- just add several sections to the json file.

Full command line options can be printed out with,

```
backup -help
```

