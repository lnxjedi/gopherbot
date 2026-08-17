---

**git-command** - *privileged*

Usage:
- `AddTask git-command clone <repoURL> <branch> <directory>` - Clones a repository from `<repoURL>` into `<directory>`. If `<branch>` is specified, it clones that branch; otherwise, it clones the default branch.
- `AddTask git-command pull <directory>` - Pulls the latest changes in the repository located at `<directory>`.
- `AddTask git-command switch <branch> <directory>` - Switches to `<branch>` in the repository at `<directory>`. If the branch does not exist locally, it creates it to track the remote branch.
- `AddTask git-command push <branch-if-no-upstream> <commit_msg> <directory>` - Commits and pushes changes in the repository at `<directory>`. If the current branch has no upstream, it pushes to `origin/<branch-if-no-upstream>` with the provided `<commit_msg>`; otherwise, it pushes to the tracked remote branch.

The `git-command` task provides essential Git operations within the pipeline, enabling actions such as cloning repositories, pulling updates, checking out branches, and pushing changes. It is designed for straightforward, non-development workflows to support regular robot operations without the complexity of full Git functionalities.

This task integrates seamlessly with the `ssh-agent` and `ssh-git-helper` tasks to manage SSH authentication and host key verification, ensuring secure interactions with Git repositories.

**See also:**
- plugins/samples/save.sh - previous job for pushing code to remote git repos
