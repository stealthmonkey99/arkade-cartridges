# Security Policy

## Reporting a vulnerability

Please report vulnerabilities **privately**, not in a public issue: use
the "Report a vulnerability" button under this repo's **Security** tab, or
email `security@arkade.build`. A seed and input log, or a minimal
cartridge that triggers the issue, helps most. Expect an acknowledgment
within about a week; fixes land on a hobby project's best effort.

## Scope

The sandbox, the host ABI, the web player and the cabinet belong to the
engine; its [SECURITY.md](https://github.com/stealthmonkey99/arkade/blob/main/SECURITY.md)
sets their scope and the disclosure terms. Here, a report is about a
cartridge doing something its sandbox should have stopped or a player
should not be able to make it do: writing outside its own storage or score
table, say, or submitting a score the game never awarded.
