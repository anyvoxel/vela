<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Review Conventions](#review-conventions)
  - [Create pull request](#create-pull-request)
  - [During code review](#during-code-review)
  - [Before merging](#before-merging)
  - [After merging](#after-merging)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Review Conventions

## Create pull request

Pull Request (PR) title **MUST** be succinct and provide necessary information. Avoid using non-descriptive
titil such as "fix some bugs", "minor changes", etc. Title is **VERY** important - we use it to generate
changelog, please think twice before creating a PR.

If your change is not ready but want to send out for early review, please send PR and prefix it with "WIP:",
which stands for "Work In Progress".

## During code review

During code reivew, assignees will request changes for correctness, improvement, etc. For each round of
code review, always create **NEW** commit for your change, i.e. do not modify existing commit. This will
dramastically help reviewers look at new changes. For example, after several round of code review, your
PR commits might look like this:

```
feat(cli): add new --debug option for all subcommands
fix style comments
rebased
fix typo
```

For mass automated fixups (e.g. automated doc formatting), use one or more commits for the changes to tooling
and a final commit to apply the fixup en masse. This makes reviews much easier. For example, you should use
two commits if your PR contains golang vendor:

```
feat(release): add initial release controller framework
feat(*): add release controller vendor
```

After pushing new changes, you **SHOULD** add comment to notify your assignees that your pull request is
ready for next round of review. Generally, you can reply with:

```
@someone comments addressed, PTAL
```

"PTAL" standards for "Please Take Another Look".

assignees should comment "/lgtm" (for reviewer) or "/approve" (for approver) if the changes are ok to be merged.

If during code review, you need to make changes not related to your orignal PR, please create a new PR
instead of appending to the original PR.

## Before merging

Before merging a PR, any temporary commits during code review needs to be squashed, for example, in the
above example, the three temporary commits will be squashed into one, i.e. from

```
feat(cli): add new --debug option for all subcommands
fix style comments
rebased
fix typo
```

to

```
feat(cli): add new --debug option for all subcommands
```

This single commit contains fixes in the three temporary commits.

## After merging

Congratulations!

Chances are that your change causes unexpected behavior and revert is needed. In such case, please create a
new PR titled "revert" and undo your changes in the PR.
