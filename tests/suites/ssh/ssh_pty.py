#!/usr/bin/env python3
# Copyright 2026 Canonical Ltd.
# Licensed under the AGPLv3, see LICENCE file for details.

"""Check a commandless SSH session through a real terminal, including resize."""

import argparse
import errno
import fcntl
import os
import pty
import select
import signal
import struct
import sys
import termios
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    checks = parser.add_mutually_exclusive_group()
    checks.add_argument("--resize", action="store_true",
                        help="also check bash COLUMNS and LINES after resize")
    checks.add_argument("--no-shell", action="store_true",
                        help="require a failed session with a no-shell error")
    parser.add_argument("--shell", choices=("bash-login", "sh"),
                        help="check the selected shell and login profile")
    parser.add_argument("--timeout", type=float, default=90)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        parser.error("supply the SSH command after --")

    child, terminal = pty.fork()
    if child == 0:
        set_size(0, 80, 24)
        os.execvp(command[0], command)

    output = bytearray()
    deadline = time.monotonic() + args.timeout
    reaped = False
    try:
        if not args.no_shell:
            # Split markers prevent an echoed input line passing a check.
            expect(terminal, output, deadline,
                   b"printf '\\nREADY_%s\\n' PTY\n", b"READY_PTY")
            expect(terminal, output, deadline,
                   b"[ -t 0 ] && printf '\\nPTY_%s\\n' OK\n", b"PTY_OK")
            if args.shell == "bash-login":
                expect(terminal, output, deadline,
                       b'test -n "$BASH_VERSION" && shopt -q login_shell && '
                       b'test "$JUJU_SSH_PROFILE" = loaded && '
                       b"printf '\\nLOGIN_%s\\n' PROFILE_OK\n",
                       b"LOGIN_PROFILE_OK")
            elif args.shell == "sh":
                expect(terminal, output, deadline,
                       b'test -z "$BASH_VERSION" && '
                       b'test -z "$JUJU_SSH_PROFILE" && '
                       b"printf '\\nSH_%s\\n' DEFAULT_OK\n",
                       b"SH_DEFAULT_OK")
            if args.resize:
                size_command = b"printf '\\nSIZE_%s_%s\\n' \"$COLUMNS\" \"$LINES\"\n"
                expect(terminal, output, deadline,
                       size_command, b"SIZE_80_24")
                set_size(terminal, 100, 40)
                # Resize traverses the client, server and Kubernetes exec
                # stream asynchronously. Re-query until it arrives.
                expect(terminal, output, deadline,
                       size_command, b"SIZE_100_40", repeat=True)
            os.write(terminal, b"exit\n")

        while read_output(terminal, output, deadline):
            pass
        status = wait_child(child, deadline)
        reaped = True
        exit_code = os.waitstatus_to_exitcode(status)
        if args.no_shell:
            if exit_code == 0:
                raise RuntimeError("shell-less SSH terminal session succeeded")
            if b"couldn't find any shell" not in output:
                raise RuntimeError("no friendly no-shell diagnostic")
            for raw_error in (b"executable file not found",
                              b"no such file or directory"):
                if raw_error in output:
                    raise RuntimeError(f"raw runtime error: {raw_error!r}")
        elif exit_code != 0:
            raise RuntimeError("SSH terminal session exited unsuccessfully")
        return 0
    except (OSError, RuntimeError, TimeoutError) as error:
        print(f"PTY check failed: {error}", file=sys.stderr)
        return 1
    finally:
        if not reaped:
            try:
                os.killpg(child, signal.SIGKILL)
            except ProcessLookupError:
                pass
            os.waitpid(child, 0)
        os.close(terminal)
        sys.stdout.buffer.write(output)
        sys.stdout.buffer.flush()


def wait_child(child, deadline):
    while True:
        waited, status = os.waitpid(child, os.WNOHANG)
        if waited:
            return status
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError("SSH process did not exit after closing its PTY")
        time.sleep(min(remaining, 0.05))


def set_size(terminal, columns, rows):
    fcntl.ioctl(terminal, termios.TIOCSWINSZ,
                struct.pack("HHHH", rows, columns, 0, 0))


def expect(terminal, output, deadline, command, marker, repeat=False):
    start = len(output)
    os.write(terminal, command)
    next_query = time.monotonic() + 0.2
    while marker not in output[start:]:
        if not read_output(terminal, output, deadline, poll=repeat):
            raise RuntimeError(f"session ended before {marker!r}")
        if repeat and time.monotonic() >= next_query:
            os.write(terminal, command)
            next_query = time.monotonic() + 0.2


def read_output(terminal, output, deadline, poll=False):
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise TimeoutError("timed out waiting for the SSH terminal session")
    ready, _, _ = select.select([terminal], [], [],
                                min(remaining, 0.2) if poll else remaining)
    if not ready:
        return True
    try:
        data = os.read(terminal, 65536)
    except OSError as error:
        if error.errno != errno.EIO:
            raise
        return False
    output.extend(data)
    return bool(data)


if __name__ == "__main__":
    sys.exit(main())
