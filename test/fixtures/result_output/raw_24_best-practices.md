---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

The **expand-contract pattern** is the recommended approach for database migrations in blue-green deployments that require backward compatibility. This pattern decouples schema changes from application deployment, ensuring both the blue (old) and green (new) environments can operate safely with the same database.

## How the Expand-Contract Pattern Works

**Expand phase**: Add new schema elements (columns, tables) to the database without removing old ones. Deploy the green environment with code that writes to both old and new columns simultaneously. This ensures the database supports both application versions during the traffic shift.

```
# Version 2 code - writes to both columns
def update_user(user_id, username):
    db.execute(
        "UPDATE users SET username = %s, user_name = %s WHERE id = %s",
        (username, username, user_id)
    )
```

**Contract phase**: Only after verifying the new version is stable (typically 24-72 hours later), remove the old schema elements. This allows sufficient time to catch issues while safer rollback strategies remain available.

## Rollback Strategies by Deployment Stage

**During traffic shift (safest option)**: Use immediate rollback to revert traffic back to blue before all traffic has moved to green. CodeDeploy automatically terminates green tasks, and blue continues functioning because the old database columns remain intact.

**After traffic shift (safer option)**: Run the rollback script to shift traffic back to blue. This works because the database still contains both old and new schema elements, so blue can immediately serve traffic with all the latest data written by green.

**Post-contract phase (last resort)**: Restore from a database snapshot taken immediately before the contract migration. This takes 10-30 minutes, causes data loss, and requires downtime—which is why the 24-72 hour waiting period is critical to catch issues first.

## Key Principles

The expand-contract pattern is recommended as default because it **maintains backward compatibility and enables safe rollbacks**. Never couple database migrations tightly with the blue-green traffic switch; schema changes should be independent and deployed through separate pipelines.