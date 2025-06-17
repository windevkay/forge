# FORGE

This repository contains multiple Go packages, each in its own subdirectory with individual `go.mod` files.

## GitHub Actions CI/CD

The repository is configured with GitHub Actions to automatically:

1. **Discover all Go packages** - Automatically finds all subdirectories containing `go.mod` files
2. **Run tests** - Executes `go test` for each package across multiple Go versions (1.21 and 1.22)
3. **Generate coverage reports** - Creates HTML coverage reports for each package
4. **Lint code** - Runs `golangci-lint` on each package
5. **Build packages** - Ensures all packages can be built successfully

### Workflow Triggers

The CI pipeline runs on:
- Push to `main`, `master`, or `develop` branches
- Pull requests targeting `main`, `master`, or `develop` branches

### Package Structure

Each Go package should be in its own directory with:
```
package-name/
├── go.mod
├── go.sum (if dependencies exist)
├── *.go files
└── *_test.go files
```

### Features

- **Parallel execution**: Tests run in parallel for each package and Go version
- **Caching**: Go modules are cached to speed up builds
- **Coverage reports**: HTML coverage reports are generated and stored as artifacts
- **Multiple Go versions**: Tests run against Go 1.21 and 1.22
- **Linting**: Code quality checks with golangci-lint
- **Flexible discovery**: Automatically adapts when new packages are added

### Configuration Files

- `.github/workflows/test.yml` - Main CI/CD pipeline
- `.golangci.yml` - Linting configuration (optional, used if present)

### Adding New Packages

To add a new Go package:
1. Create a new directory
2. Run `go mod init your-package-name` in that directory
3. Add your Go code and tests
4. Push to GitHub - the CI will automatically detect and test the new package

### Viewing Results

- Test results are visible in the GitHub Actions tab
- Coverage reports are available as downloadable artifacts
- Failed tests will block PRs if configured as required status checks

