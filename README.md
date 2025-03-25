# Panopticon

A TUI command runner with a built-in file-watcher.
![panopticon](https://github.com/user-attachments/assets/b86a80da-6e4e-4d2d-9d15-74ea84fd4f1c)

## Features
- Configurable command runner with specified file paths to watch recursively
- View output from commands after running with scrollable viewport

## Installation
### Via `go install` (recommended)
With go 1.23 or higher:
```
go install github.com/cfbender/panopticon@latest
```

### From source
Clone the repo and run `go install`.

If you don't have a config file, you can run `panopticon init` to generate one at your current working directory.

## Usage
Given the following config:
```yaml
commands:
  - cmd: echo "components"
    watch_paths:
      - ./src/components
  - cmd: echo "source"
    watch_paths:
      - ./src
```
and a directory structure looking like
```
src/
|_ components/
|__|__ some-component.tsx
|__ index.ts
package.json
```

- Start the watcher with no commands running
```sh
panopticon
```
Changing the file `index.ts` would run only `echo "source"`, where changing `src/components/some-component.tsx` would run both `echo "source"` and `echo "components"`.

### TUI commands

- `h/j` or `up/down` to navigate between commands
- `enter` to view the output
- `ctrl+j/ctrl+k`/`ctrl+up/ctrl+down` to navigate output in viewport
- `r` to run command immediately

### Options

- `--help` or `-h`
```sh
panopticon -h
```
Outputs the various flags that can be passed
- `--run-on-start` or `-r`
```sh
panopticon --run-on-start
```
Will run all commands currently upon watcher start, and then again on subsequent changes.

- `--match` or `-m`
```sh
panopticon --match "*echo*"
```
Will run all commands matching the glob pattern `*echo*`

- `--version` or `-v`
```sh
panopticon --version
```
Will print the version of `panopticon`
