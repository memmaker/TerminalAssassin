# AGENTS.md — Terminal Assassin

A terminal-aesthetic stealth/assassination roguelite written in Go, rendered via a custom Unicode cell grid on top of [Ebiten v2](https://ebitengine.org/).

## For AI Agents: Simplicity rules

When implementing features, always look for the simplest possible solution that achieves the desired outcome. Avoid over-engineering or adding unnecessary complexity.

Prefer solutions which are easy to understand & maintain.

Favor the least amount of code possible.

## For AI Agents: Ask me

If the user prompt is not clear, ask for clarification. Do not make assumptions about what the user wants.

If any user prompts leads to massive refactoring or a large amount of code, point out the risks, and ask for confirmation before proceeding. Suggest better alternatives if you can think of any.

If any action the user wants seems counterintuitive or non-sensical, ask for confirmation before proceeding. Suggest better alternatives if you can think of any.

## For AI Agents: No scripts, no python, no shell

AI Agents should not utilize external scripts or shell commands or python.

AI Agents should use the internal file creation and manipulation capabilities of the IDE and their LLM integration.

AI Agents should never use heredoc for anything.

The only exception are compile error checks and GIT Operations.

## For AI Agents: Be brief and concise

Always use caveman mode (skill).
