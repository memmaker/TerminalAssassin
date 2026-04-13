# AGENTS.md — Terminal Assassin

A terminal-aesthetic stealth/assassination roguelite written in Go, rendered via a custom Unicode cell grid on top of [Ebiten v2](https://ebitengine.org/).

## For AI Agents: No scripts, no python, no shell

AI Agents should not utilize external scripts or shell commands or python.

AI Agents should use the internal file creation and manipulation capabilities of the IDE and their LLM integration.

AI Agents should never use heredoc for anything.

The only exception are compile error checks.