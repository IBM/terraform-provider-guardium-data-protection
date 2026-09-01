# Automated Release Guide

This repository uses **semantic-release** for automated versioning and releases.

## How It Works

When you merge a PR to the `main` branch, the system automatically:
1. Analyzes the PR title
2. Determines the version bump (major, minor, or patch)
3. Generates a changelog
4. Creates a GitHub release with a tag
5. Triggers the existing release workflow to build and publish

## Version Bumping Rules

### Three Simple Rules Based on PR Title:

| PR Title Contains | Version Bump | Example |
|-------------------|--------------|---------|
| **"major"** or **"MAJOR"** | **Major** (1.3.0 → 2.0.0) | "Major: Redesign authentication API" |
| **"patch"**, **"fix"**, or **"Fix"** | **Patch** (1.3.0 → 1.3.1) | "Fix authentication timeout", "Patch security issue" |
| **Anything else** | **Minor** (1.3.0 → 1.4.0) | "Add new feature", "Update documentation" |

## Examples

### Patch Release (Bug Fixes)
**PR Title:** "Fix authentication timeout issue"
**Result:** 1.3.0 → 1.3.1

**PR Title:** "Patch: Resolve memory leak in client"
**Result:** 1.3.0 → 1.3.1

**PR Title:** "Quick fix for nil pointer error"
**Result:** 1.3.0 → 1.3.1

### Minor Release (New Features - Default)
**PR Title:** "Add support for VA notifications"
**Result:** 1.3.0 → 1.4.0

**PR Title:** "Implement new datasource configuration"
**Result:** 1.3.0 → 1.4.0

**PR Title:** "Update documentation and examples"
**Result:** 1.3.0 → 1.4.0

### Major Release (Breaking Changes)
**PR Title:** "Major: Redesign authentication API"
**Result:** 1.3.0 → 2.0.0

**PR Title:** "MAJOR refactor of provider configuration"
**Result:** 1.3.0 → 2.0.0

**PR Title:** "Breaking changes - major update to datasource schema"
**Result:** 1.3.0 → 2.0.0

## When to Use Each Type

### Use **Patch** (1.3.0 → 1.3.1) for:
- Bug fixes
- Security patches
- Performance improvements
- Documentation fixes
- Small corrections

### Use **Minor** (1.3.0 → 1.4.0) for:
- New features
- New resources or data sources
- Enhancements to existing features
- Deprecations (with backward compatibility)
- Most changes (this is the default)

### Use **Major** (1.3.0 → 2.0.0) for:
- Breaking changes
- Removing deprecated features
- Changing existing behavior
- Incompatible API changes
- Requiring user action to upgrade

## Workflow

1. **Create a branch** for your changes
2. **Make your commits** (any format is fine)
3. **Create a PR** with a descriptive title
4. **Add keyword to PR title** based on change type:
   - Add "major" for breaking changes
   - Add "fix" or "patch" for bug fixes
   - Use any other title for new features (default to minor)
5. **Merge the PR** (squash or merge, both work)
6. **Automated release runs** on main branch
7. **Version is determined** from PR title
8. **Tag is created** (e.g., `v1.4.0`)
9. **Changelog is updated** automatically
10. **GitHub release is created** with release notes
11. **Release workflow triggers** to build and publish

## Current Version

Latest version: **v1.3.0**

Next release will be:
- **v2.0.0** (if PR title contains "major")
- **v1.3.1** (if PR title contains "fix" or "patch")
- **v1.4.0** (default for everything else)

## Skipping Releases

If you don't want to trigger a release, add `[skip ci]` to your PR title or commit message.

## Manual Override

If you need to manually create a release:
1. Create a tag manually: `git tag v1.4.0`
2. Push the tag: `git push origin v1.4.0`
3. The existing release workflow will trigger

## Troubleshooting

### No release was created
- Check GitHub Actions logs for errors
- Ensure the PR was merged to `main` branch
- Verify the workflow file exists and is enabled

### Wrong version bump
- Check PR title for keywords: "major", "fix", "patch"
- Default is always minor bump if no keywords found

### Release failed
- Check GPG secrets are configured: `GPG_PRIVATE_KEY` and `PASSPHRASE`
- Verify GitHub token has write permissions
- Review workflow logs in GitHub Actions

## Best Practices

1. **Use descriptive PR titles** - They become part of your changelog
2. **Mark changes clearly** - Use "major", "fix", or "patch" in PR title
3. **Review before merging** - Ensure the PR title reflects the change type
4. **One logical change per PR** - Makes version bumping predictable
5. **Document breaking changes** - Explain what changed and how to migrate

## Configuration Files

- `.releaserc.json` - Semantic-release configuration (three-tier rules)
- `.github/workflows/auto-release.yml` - Automated release workflow
- `.github/workflows/release.yml` - Build and publish workflow (triggered by tags)

## Questions?

Contact the maintainers or open an issue for help with the release process.