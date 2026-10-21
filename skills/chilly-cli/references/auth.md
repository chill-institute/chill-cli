# Auth Reference

`chilly auth login` prints the hosted `/auth/chilly-token` page, then waits for
the setup token at a hidden prompt. The page can be opened in any signed-in
browser, including on another machine. Remote API hosts must use HTTPS; HTTP is
accepted only for loopback development.

## Commands

- Interactive login: `chilly auth login`
- Preview the token page URL: `chilly auth login --dry-run --output json`
- Token from stdin: `printf '{"token":"token-from-setup","skip_verify":true}' | chilly auth login --json @- --dry-run --output json`
- Token as an argument, only when stdin is unavailable (visible to other processes): `chilly auth login --token <token>`
- Localhost callback: `chilly auth login --local-browser`, add `--no-browser` to skip auto-open
- Logout: `chilly auth logout --output json`, previewable with `--dry-run`
- Confirm the token works: `chilly whoami --output json`
- Auth and host together: `chilly doctor --fields auth.status,config.api_base_url,config.profile --output json`

## Host and Profile

- Config path: `chilly settings path --fields path --output json`
- Profile and host: `chilly settings show --fields profile,api_base_url --output json`
- Isolated profile path: `chilly settings path --profile dev --output json`
