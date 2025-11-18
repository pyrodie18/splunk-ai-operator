---
name: Dependency Updates Tracking
about: Track batch dependency updates for a release
title: 'Dependency Updates for v[VERSION]'
labels: dependencies, enhancement
assignees: ''

---

## Overview

Track dependency updates to be included in release v[VERSION].

## High Priority Updates

These should be reviewed and updated soon:

### Kubernetes Core Dependencies
- [ ] `k8s.io/api`: [current] → [target]
- [ ] `k8s.io/apimachinery`: [current] → [target]
- [ ] `k8s.io/apiextensions-apiserver`: [current] → [target]
- [ ] `k8s.io/client-go`: [current] → [target]

**Impact:** Core Kubernetes compatibility
**Testing Required:** Full operator test suite, E2E tests

### Controller Runtime
- [ ] `sigs.k8s.io/controller-runtime`: [current] → [target]

**Impact:** Operator framework updates
**Testing Required:** Controller reconciliation, webhook functionality

### Certificate Manager
- [ ] `github.com/cert-manager/cert-manager`: [current] → [target]

**Impact:** Certificate management functionality
**Testing Required:** TLS certificate generation, webhook certificates

## Medium Priority Updates

Can be included if time permits:

### Cloud Provider SDKs
- [ ] `github.com/Azure/azure-sdk-for-go/sdk/azidentity`: [current] → [target]
- [ ] `github.com/aws/aws-sdk-go-v2/*`: [current] → [target]

**Impact:** Cloud provider integration
**Testing Required:** Cloud-specific features (if any)

### Observability
- [ ] `github.com/prometheus/client_golang`: [current] → [target]
- [ ] `sigs.k8s.io/yaml`: [current] → [target]

**Impact:** Metrics and configuration parsing
**Testing Required:** Metrics endpoint, YAML parsing tests

## Low Priority Updates

Nice to have, but not critical:

### Testing Dependencies
- [ ] `github.com/stretchr/testify`: [current] → [target]
- [ ] `github.com/go-logr/logr`: [current] → [target]

**Impact:** Test functionality only
**Testing Required:** Existing test suite

## Update Process

### 1. Create Update Branch
```bash
git checkout -b chore/dependency-updates-v[VERSION]
```

### 2. Update Go Modules
```bash
# Update all dependencies
go get -u ./...

# Or update specific packages
go get -u k8s.io/api@v0.34.2
go get -u k8s.io/apimachinery@v0.34.2
# ... etc

# Tidy up
go mod tidy
```

### 3. Test Changes
```bash
# Run unit tests
make test

# Run lint
make lint

# Build operator
make build

# Test deployment (local cluster)
make deploy IMG=controller:latest
```

### 4. Update Compatibility Matrix
Update `compatibility-matrix.yaml` with new versions:
```yaml
platform:
  kubernetes:
    minVersion: "X.Y.Z"  # Update if changed
    maxVersion: "X.Y.Z"  # Update if changed
```

### 5. Generate New BOM
```bash
make generate-bom VERSION=[VERSION]
```

### 6. Create PR
- Reference this tracking issue
- Include test results
- Note any breaking changes

## Testing Checklist

Before merging:

- [ ] All unit tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Operator builds successfully (`make build`)
- [ ] Operator deploys to test cluster
- [ ] Sample CRDs can be created
- [ ] Webhooks function correctly
- [ ] No regression in existing functionality
- [ ] Compatibility matrix updated
- [ ] BOM regenerated

## Breaking Changes

Document any breaking changes here:

- None expected (minor/patch updates only)

## References

- Dependabot configuration: `.github/dependabot.yml`
- Related Dependabot issues: #[numbers]
- Testing guide: `docs/testing.md`

## Notes

Add any additional notes or concerns here.
