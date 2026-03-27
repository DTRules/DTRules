# DTRules v5.0 Release Guide

This guide covers releasing DTRules v5.0 to Maven Central.

## Prerequisites

1. **Maven 3.6+** installed
2. **GPG key** for signing artifacts
3. **Sonatype OSSRH account** with access to `com.dtrules` group
4. **~/.m2/settings.xml** configured with credentials

## Step 1: Configure Maven Settings

Create or update `~/.m2/settings.xml` with your credentials:

```xml
<settings>
  <servers>
    <server>
      <id>sonatype-nexus-snapshots</id>
      <username>YOUR_SONATYPE_USERNAME</username>
      <password>YOUR_SONATYPE_PASSWORD</password>
    </server>
    <server>
      <id>sonatype-nexus-staging</id>
      <username>YOUR_SONATYPE_USERNAME</username>
      <password>YOUR_SONATYPE_PASSWORD</password>
    </server>
  </servers>
  <profiles>
    <profile>
      <id>gpg</id>
      <properties>
        <gpg.executable>gpg</gpg.executable>
        <gpg.passphrase>YOUR_GPG_PASSPHRASE</gpg.passphrase>
      </properties>
    </profile>
  </profiles>
  <activeProfiles>
    <activeProfile>gpg</activeProfile>
  </activeProfiles>
</settings>
```

## Step 2: Verify GPG Key

```bash
# List your GPG keys
gpg --list-secret-keys --keyid-format LONG

# If you need to create a new key:
gpg --gen-key

# Publish your public key to a keyserver (required for Maven Central)
gpg --keyserver keyserver.ubuntu.com --send-keys YOUR_KEY_ID
```

## Step 3: Verify Build

```bash
# Clean build and test
mvn clean install

# Verify all modules compile
mvn clean compile -DskipTests
```

## Step 4: Prepare Release

Option A: Using Maven Release Plugin (recommended)
```bash
# Prepare the release (updates versions, creates tags)
mvn release:prepare -DdryRun=true  # Test first
mvn release:prepare                 # Actually prepare

# Perform the release (builds, signs, deploys)
mvn release:perform
```

Option B: Manual Release
```bash
# Update version from SNAPSHOT to release
mvn versions:set -DnewVersion=5.0
mvn versions:commit

# Build with signing
mvn clean deploy -DperformRelease=true -Dgpg.passphrase=YOUR_PASSPHRASE

# After successful deploy, tag the release
git tag -a v5.0 -m "Release v5.0"
git push origin v5.0

# Update to next SNAPSHOT
mvn versions:set -DnewVersion=5.1-SNAPSHOT
mvn versions:commit
git add -A && git commit -m "Prepare for next development iteration"
git push
```

## Step 5: Release on Sonatype

1. Log in to https://oss.sonatype.org
2. Go to "Staging Repositories"
3. Find your `comdtrules-XXXX` repository
4. Click "Close" and wait for validation
5. If validation passes, click "Release"
6. Artifacts will sync to Maven Central within 2 hours

## Step 6: Create GitHub Release

```bash
# Create GitHub release with release notes
gh release create v5.0 --title "DTRules v5.0" --notes "See CHANGELOG.md for details"
```

## Troubleshooting

### GPG signing fails
- Ensure gpg-agent is running: `gpgconf --launch gpg-agent`
- Check key permissions: `gpg --list-secret-keys`
- Try with explicit passphrase: `-Dgpg.passphrase=...`

### Sonatype validation fails
- Check all POMs have: name, description, url, license, developers, scm
- Ensure sources and javadoc JARs are being generated
- Verify GPG signatures are present

### "Unauthorized" errors
- Verify settings.xml server IDs match repository IDs in POM
- Check Sonatype credentials are correct
- Ensure you have deploy permissions for com.dtrules group

## Release Checklist

- [ ] All tests pass: `mvn test`
- [ ] Build succeeds: `mvn clean install`
- [ ] GPG key published to keyserver
- [ ] settings.xml configured with credentials
- [ ] CHANGELOG.md updated with v5.0 changes
- [ ] Version updated from 5.0-SNAPSHOT to 5.0
- [ ] Artifacts deployed to Sonatype staging
- [ ] Staging repository closed and released
- [ ] Git tag created and pushed
- [ ] GitHub release created
- [ ] Version bumped to 5.1-SNAPSHOT for next development
