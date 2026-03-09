# Playground

This directory is for **End-to-End (E2E) testing** and integration debugging of Orion.

## Contents

- `test_e2e.sh`: Automated E2E test script.
- `Orion_workspace/`: Temporary workspace generated during tests (gitignored).

## Running E2E Tests

The `test_e2e.sh` script automates the full workflow:

1.  **Rebuilds & Installs** the latest `orion` binary.
2.  **Initializes** a workspace from `markshao/Orion`.
3.  **Spawns** a node (`test-node-1`).
4.  **Simulates** development (creates/commits a file).
5.  **Merges** the node back to the logical branch.
6.  **Verifies** the merge result.

### Usage

Run from the project root:

```bash
./playground/test_e2e.sh
```

Or from within this directory:

```bash
cd playground
./test_e2e.sh
```

### Note

- The script **deletes** `playground/Orion_workspace` before running. Do not store important files there.
- All git operations (clone, commit, merge) are performed locally in the temporary workspace.
