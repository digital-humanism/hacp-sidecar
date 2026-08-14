#!/usr/bin/env python3
"""Temporary script: remove multi-line log.Printf("[DEBUG ...") blocks."""

import sys

path = r"cmd\hacp-conformance-runner\main.go"

with open(path, "r", encoding="utf-8") as f:
    lines = f.readlines()

out = []
i = 0
n = len(lines)
removed = 0

while i < n:
    stripped = lines[i].strip()

    # Case 1: single-line  log.Printf("[DEBUG ...")
    if stripped.startswith('log.Printf("[DEBUG'):
        i += 1
        removed += 1
        continue

    # Case 2: multi-line  log.Printf(  +  next line starts with "[DEBUG
    if stripped == "log.Printf(" and i + 1 < n and lines[i + 1].strip().startswith('"[DEBUG'):
        i += 1  # skip log.Printf(
        # skip until closing paren line
        while i < n:
            s = lines[i].strip()
            i += 1
            if s in (")", "),"):
                break
        removed += 1
        continue

    out.append(lines[i])
    i += 1

with open(path, "w", encoding="utf-8") as f:
    f.writelines(out)

print(f"Removed {removed} debug log blocks")