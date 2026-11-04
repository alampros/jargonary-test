# Project instructions

## Required completion gate

Before considering any change completed, run **`go tool task check`** from the repository root and ensure it passes. Run it again after fixing failures or changing code/configuration. Do not skip checks or weaken their configuration to claim success. If the gate cannot run, report the concrete blocker and do not describe the change as complete.

The gate runs Go formatting (`go fmt`, failing if it changes packages), `go vet`, Go unit/integration tests with the race detector, Biome lint/format/import checks, TypeScript checking, and Vitest UI unit tests. Formatting changes made by the gate must be reviewed before rerunning it.

Task is a project-local Go tool, pinned in `go.mod`; invoke it with `go tool task`. Biome and Vitest are project-local npm dependencies in `web`. Use `go tool task --list` for the other tasks. Do not reintroduce Make.

Air is also pinned as a project-local Go tool. Use `go tool task dev` to run Air for Go rebuilds and Vite for React HMR. Air excludes the web source tree and generated assets; Go serves its normal HTML with Vite modules only when `-dev-ui` is enabled. Keep Air out of the runtime container.

## System software and package managers

NEVER install, upgrade, uninstall, or modify system-level software.

Do not run Homebrew, apt, yum, dnf, pacman, snap, MacPorts, or equivalent package-manager commands, even when a task would benefit from another tool.

Do not request tool escalation for these commands. If a required program is missing, stop and tell the user what is missing. Wait for the user to install it. This rule applies even when an automated approval system would permit the command.

Project-scoped dependency management, such as the Go tool directive and npm dependencies in `web`, must not install global binaries or modify system software.
