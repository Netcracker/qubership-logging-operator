# Maven — GitHub Packages auth (local build)

Many Qubership Java monorepos resolve BOMs and internal artifacts from **GitHub Packages**
(`https://maven.pkg.github.com/...`). A **401 Unauthorized** during `mvn compile` is often **misconfigured local Maven
auth**, not a logging-change failure.

## Symptom

```text
Could not transfer artifact ... from/to github (https://maven.pkg.github.com/...): status code: 401, reason phrase: Unauthorized
```

Build and smoke gates stay **blocked** until this is fixed or CI evidence is used — do not treat 401 as “Java migration
impossible” without checking auth.

## Fix (local)

1. **Read the repo `pom.xml`** (or parent POM) for the `<repository>` / `<server>` **id** — commonly `github`, but use
   the exact id from the target repo.

   ```xml
   <repository>
     <id>github</id>
     <url>https://maven.pkg.github.com/netcracker/...</url>
   </repository>
   ```

2. **Add matching credentials** in `~/.m2/settings.xml` (create the file if missing). The `<server><id>` must match the
   POM id **exactly**:

   ```xml
   <settings xmlns="http://maven.apache.org/SETTINGS/1.2.0"
             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
             xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0
               https://maven.apache.org/xsd/settings-1.2.0.xsd">
     <servers>
       <server>
         <id>github</id>
         <username><!-- GitHub username --></username>
         <password><!-- PAT with read:packages (and repo if needed) --></password>
       </server>
     </servers>
   </settings>
   ```

3. **Personal access token (PAT):** GitHub → Settings → Developer settings → Personal access tokens. Grant
   **`read:packages`**; for private org repos, the account must have access to the org/package. Fine-grained tokens:
   Packages read on the relevant org/repos.

4. **Re-run** the repo-documented build (e.g. `mvn -pl <module> -am compile`).

5. **Record in the migration report** if auth was missing and then fixed — helps the next contributor.

## Agent behavior

- On **401 from `maven.pkg.github.com`**, check POM repository id vs `settings.xml` and **ask the user** to configure
  PAT/server entry before marking Java build permanently blocked.
- Do **not** commit PATs or paste tokens into the report/PR.
- CI on the PR often already has `GITHUB_TOKEN` / org secrets — local setup is the usual gap for outside contributors
  and fresh machines.

## Still blocked after auth?

- Wrong `<id>` (typo vs POM), expired PAT, no org package permission, or VPN/network policy — capture the **exact** Maven
  error in **Blocked validation**.
- Multiple repository ids in the POM — add one `<server>` block per id.
