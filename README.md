# claudeusage

Prints your weekly Claude subscription usage: all models, and Fable.

```
$ claudeusage
Weekly   67%  resets Fri 6 Nov 21:00 (in 7h54m)
Fable   100%  resets Fri 6 Nov 21:00 (in 7h54m)
```

`claudeusage -json` prints the raw response.

## Install

```sh
go install github.com/maragubot/claudeusage@latest
```

## How it works

It reads the OAuth access token that Claude Code stores. On macOS this is the login Keychain item `Claude Code-credentials`, and the fallback is `~/.claude/.credentials.json`. It calls `GET https://api.anthropic.com/api/oauth/usage` with that token.

- It only reads the credentials. It never refreshes the token or writes it anywhere. If the token has expired, run Claude Code once.
- It sends the token only to api.anthropic.com over HTTPS and does not follow redirects.
- The endpoint is undocumented and can change without notice.
