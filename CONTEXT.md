# Yontrack CLI

The command line client for Yontrack. This file records what the words in this
repository mean; it is a glossary, not a specification.

## Language

**Yontrack**:
The product, and the server this CLI talks to. Currently at version 5.
_Avoid_: Ontrack, the server, the instance, the platform

**Yontrack CLI**:
The client in this repository. What a user installs and runs.
_Avoid_: Ontrack CLI, the tool, the client

**`yontrack`**:
The binary, and the command a user types.
_Avoid_: ontrack-cli, the executable

**Ontrack**:
The retired name of the product. It survives only where changing it would break
something: the `nemerosa/ontrack` GitHub organisation, the companion repository
names, the `ontrack.graphql` schema file, the `X-Ontrack-Token` header, and the
`--ontrack` flag on `version`. Those are identifiers, not names — do not rename
them. Anywhere the word is prose, it is wrong.

**`ontrack-cli`**:
The retired name of the binary. No release from 5.4.0 onwards publishes it. It
survives as the on-disk name the v4 GitHub Action gives the file it downloads.
