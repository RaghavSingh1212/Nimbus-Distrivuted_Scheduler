# Debug Fix Applied

## Problem Identified
Tasks were being created and stored in the database, but workers weren't picking them up because:
- The scheduler only checked an in-memory priority queue
- If the master restarted, the queue was empty
- Pending tasks in the database were never loaded into the queue

## Fix Applied
1. Added `loadPendingTasks()` function to load pending tasks from DB into queue
2. Call it when scheduler initializes (on master startup)
3. Also call it in `PollTask` if queue is empty (defensive check)

## To Test the Fix

1. **Restart the master** (the fix is in the new binary):
   ```bash
   # Stop current master (Ctrl+C)
   ./bin/master
   ```

2. **The master will now load existing pending tasks on startup**

3. **Worker should immediately pick up tasks**

4. **Test with a new task**:
   ```bash
   ./bin/client submit echo "Test after fix"
   ./bin/client list
   ```

The existing 3 pending tasks should now be processed!

