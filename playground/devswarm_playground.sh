#!/bin/bash
set -e

# Configuration
REPO_URL="https://github.com/markshao/DevSwarm.git"
WORKSPACE_NAME="DevSwarm_workspace"
LOGICAL_BRANCH="feature/e2e-test"
NODE_NAME="test-node-1"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🐝 Starting End-to-End Test for DevSwarm...${NC}"

# Ensure devswarm is installed/updated
echo -e "${BLUE}📦 Rebuilding and installing DevSwarm...${NC}"
cd ..
./install.sh
cd playground

# 1. Clean up previous run
if [ -d "$WORKSPACE_NAME" ]; then
    echo -e "${BLUE}🧹 Cleaning up previous workspace...${NC}"
    rm -rf "$WORKSPACE_NAME"
fi

# 2. Initialize Workspace
echo -e "${BLUE}🚀 Initializing workspace for $REPO_URL...${NC}"
devswarm init "$REPO_URL"

# 3. Enter Workspace
cd "$WORKSPACE_NAME"
echo -e "${GREEN}✅ Entered workspace: $(pwd)${NC}"

# 4. Spawn a Node
echo -e "${BLUE}🌱 Spawning node '$NODE_NAME' on branch '$LOGICAL_BRANCH'...${NC}"
# We use 'main' as base since feature/e2e-test doesn't exist remotely
devswarm spawn "$LOGICAL_BRANCH" "$NODE_NAME" --base main

# 5. List Nodes
echo -e "${BLUE}📋 Listing nodes...${NC}"
devswarm ls

# 6. Verify Worktree
WORKTREE_PATH="nodes/$NODE_NAME"
if [ -d "$WORKTREE_PATH" ]; then
    echo -e "${GREEN}✅ Worktree created at $WORKTREE_PATH${NC}"
else
    echo -e "${RED}❌ Worktree not found!${NC}"
    exit 1
fi

# 7. Simulate work (create a file in the node)
echo -e "${BLUE}✍️ Simulating work in node...${NC}"
TEST_FILE="$WORKTREE_PATH/test_artifact.txt"
echo "Hello from DevSwarm E2E" > "$TEST_FILE"
echo -e "${GREEN}✅ Created test artifact: $TEST_FILE${NC}"

# 8. Commit changes in the node (Simulate user action)
echo -e "${BLUE}💾 Committing changes in shadow branch...${NC}"
(
    cd "$WORKTREE_PATH"
    git add .
    git commit -m "feat: add test artifact from e2e script"
)

# 9. Merge Node
echo -e "${BLUE}🔀 Merging node back to logical branch...${NC}"
devswarm merge "$NODE_NAME" --cleanup

# 10. Verify Merge
echo -e "${BLUE}🔍 Verifying merge in repo...${NC}"
cd repo
git checkout "$LOGICAL_BRANCH"
if [ -f "test_artifact.txt" ]; then
    echo -e "${GREEN}✅ Merge successful! Artifact found in logical branch.${NC}"
else
    echo -e "${RED}❌ Merge failed! Artifact not found.${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 E2E Test Completed Successfully!${NC}"
