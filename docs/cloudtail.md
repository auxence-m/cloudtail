## cloudtail

cloudtail displays or tail logs from Google Cloud Logging

### Synopsis

cloudtail is a lightweight cloud-native command-line tool written in Golang that allows users to view or tail logs from Google Cloud Logging (similar to `kubectl logs`).

It connects to the Google Cloud Logging API, fetches logs for a specific project based on filters like severity, resource type, or time range. 

It displays the logs or continuously streams them to the terminal or to an output file in near real-time.

### Options

```
  -h, --help     help for cloudtail
  -t, --toggle   help message for toggle
```

### SEE ALSO

* [cloudtail completion](cloudtail_completion.md)	 - Generate the autocompletion script for the specified shell
* [cloudtail docs](cloudtail_docs.md)	 - Generate documentation for cloudtail
* [cloudtail tail](cloudtail_tail.md)	 - Display and stream Google Cloud Logging entries matching the specified filters

