## Development

This project uses a hybrid Go + Node.js architecture:

```bash
# Install dependencies
npm install -g pnpm
curl https://mise.run | sh
mise install

# Build Go binary
pnpm run build

# Run tests
pnpm test

# Development mode
pnpm run dev
```

## PR conventions

When creating a PR, use a title like:
```
release: patch - fix bug #123
release: minor - add new feature #456
release: major - breaking changes #789
```
The `release: ${VERSION}` part of the title is used by GitHub Actions to detect which version to bump.
