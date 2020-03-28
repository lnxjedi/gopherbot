___

**email-log** - *privileged*

Usage: `FinalTask email-log joe@example.com emily@example.com`

Normally used in the **fail** and **final** pipelines, emails a copy of the pipeline log. Can also be used with `AddTask` in the main pipeline, but content will be incomplete.

*Notes on the current implementation:*  
The email function call takes a byte slice for the email body, rather than an `io.Reader`, meaning the entire body of the message is read in to memory. To limit memory use, **Gopherbot** allocates a 10MB line-based circular buffer, with maximum 16KB-long lines (terminated by `\n`), and reads the log to that buffer for emailing. Logs that are longer than 10MB will only send the last 10MB of the log, and lines longer than 16KB will be truncated.