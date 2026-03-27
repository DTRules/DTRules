#!/bin/bash
# DTRules Release Script
# This script automates the Maven Central release process.

set -e

VERSION="5.0"
NEXT_VERSION="5.1-SNAPSHOT"

echo "========================================"
echo "DTRules Release Script v${VERSION}"
echo "========================================"
echo ""

# Check prerequisites
echo "Checking prerequisites..."

# Check Maven
if ! command -v mvn &> /dev/null; then
    echo "ERROR: Maven is not installed"
    exit 1
fi
echo "  Maven: $(mvn --version | head -1)"

# Check GPG
if ! command -v gpg &> /dev/null; then
    echo "ERROR: GPG is not installed"
    exit 1
fi
echo "  GPG: $(gpg --version | head -1)"

# Check for GPG keys
if ! gpg --list-secret-keys 2>/dev/null | grep -q "sec"; then
    echo "ERROR: No GPG secret keys found"
    echo "  Run: gpg --gen-key"
    exit 1
fi
echo "  GPG keys: Found"

# Check settings.xml
if [ ! -f ~/.m2/settings.xml ]; then
    echo "ERROR: ~/.m2/settings.xml not found"
    echo "  Copy maven-settings-template.xml to ~/.m2/settings.xml and configure"
    exit 1
fi
echo "  Maven settings: Found"

echo ""
echo "Prerequisites OK"
echo ""

# Menu
echo "Select operation:"
echo "  1) Dry run (test release without deploying)"
echo "  2) Prepare release (update versions, tag)"
echo "  3) Perform release (build, sign, deploy)"
echo "  4) Full release (prepare + perform)"
echo "  5) Cancel/rollback release"
echo "  q) Quit"
echo ""
read -p "Choice: " choice

case $choice in
    1)
        echo "Running release dry run..."
        mvn release:prepare -DdryRun=true
        echo ""
        echo "Dry run complete. Review changes in pom.xml.next files."
        mvn release:clean
        ;;

    2)
        echo "Preparing release..."
        mvn release:clean
        mvn release:prepare \
            -DreleaseVersion=${VERSION} \
            -DdevelopmentVersion=${NEXT_VERSION} \
            -Dtag=v${VERSION}
        echo ""
        echo "Release prepared. Run option 3 to deploy."
        ;;

    3)
        echo "Performing release (building, signing, deploying)..."
        mvn release:perform
        echo ""
        echo "Release deployed to Sonatype staging."
        echo "Next steps:"
        echo "  1. Log in to https://oss.sonatype.org"
        echo "  2. Go to 'Staging Repositories'"
        echo "  3. Find comdtrules-XXXX repository"
        echo "  4. Click 'Close' and wait for validation"
        echo "  5. Click 'Release' to publish to Maven Central"
        ;;

    4)
        echo "Full release (prepare + perform)..."
        mvn release:clean
        mvn release:prepare \
            -DreleaseVersion=${VERSION} \
            -DdevelopmentVersion=${NEXT_VERSION} \
            -Dtag=v${VERSION}
        mvn release:perform
        echo ""
        echo "Release complete. See option 3 for Sonatype steps."
        ;;

    5)
        echo "Rolling back release..."
        mvn release:rollback
        mvn release:clean
        echo "Rollback complete."
        ;;

    q|Q)
        echo "Exiting."
        exit 0
        ;;

    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "Done."
