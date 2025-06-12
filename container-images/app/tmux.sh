#!/usr/bin/env sh

set -o errexit
set -o nounset
# set -o xtrace

if set +o | grep -F 'set +o pipefail' > /dev/null; then
  # shellcheck disable=SC3040
  set -o pipefail
fi

if set +o | grep -F 'set +o posix' > /dev/null; then
  # shellcheck disable=SC3040
  set -o posix
fi

# -----------------------------------------------------------------------------

# inspiration:
#   https://serverfault.com/questions/1096897/how-to-start-tmux-session-with-2-panes-and-execute-in-each-pane-predefined-comma
#   https://askubuntu.com/questions/830484/how-to-start-tmux-with-several-panes-open-at-the-same-time
#   https://stackoverflow.com/questions/5609192/how-do-i-set-tmux-to-open-specified-windows-at-startup
#   https://man7.org/linux/man-pages/man1/tmux.1.html

CMD1=${1:?"first parameter, which is expected to be a command, is not set"}
CMD2=${2:?"second parameter, which is expected to be a command, is not set"}

# tmux new-session -d -s my-session-name 'watch -n 5 ps';
tmux new-session -d -s my-session-name "${CMD1}";

# tmux split-window -h 'watch -n 5 ls';
tmux split-window -h "${CMD2}";

# Use this to connect whenever you want 
tmux -2 attach-session -t my-session-name;
