## pachctl finish commit

Finish a started commit.

### Synopsis


Finish a started commit. Commit-id must be a writeable commit.

```
pachctl finish commit <repo>@<branch-or-commit>
```

### Options

```
      --description string   A description of this commit's contents (synonym for --message)
  -m, --message string       A description of this commit's contents (overwrites any existing commit description)
```

### Options inherited from parent commands

```
      --no-metrics           Don't report user metrics for this command
      --no-port-forwarding   Disable implicit port forwarding
  -v, --verbose              Output verbose logs
```

