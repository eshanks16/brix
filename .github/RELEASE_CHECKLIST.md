# Release Checklist - v1.0.0

## Pre-Release

- [x] Update version in deployment files
  - [x] `deployment/k8s/brix/04-deployment.yaml` - Changed image tag to `v1.0.0`
  - [x] Changed `imagePullPolicy` to `IfNotPresent`

- [ ] Run all tests
  ```bash
  make test
  make test-coverage
  ```

- [ ] Verify documentation is up to date
  - [ ] README.md has Kubernetes deployment section
  - [ ] deployment/README.md is accurate
  - [ ] docs/API_EXAMPLES.md is complete

- [ ] Clean build
  ```bash
  make clean
  make build
  ./brix-server
  # Verify app runs at http://localhost:8080
  ```

## Docker Image

- [ ] Build and tag Docker image
  ```bash
  cd deployment
  docker build -t eshanks16/brix-pizza:v1.0.0 -f Dockerfile ..
  docker tag eshanks16/brix-pizza:v1.0.0 eshanks16/brix-pizza:latest
  ```

- [ ] Test Docker image locally
  ```bash
  docker run -p 8080:8080 eshanks16/brix-pizza:v1.0.0
  # Verify app runs at http://localhost:8080
  ```

- [ ] Push to Docker Hub
  ```bash
  docker push eshanks16/brix-pizza:v1.0.0
  docker push eshanks16/brix-pizza:latest
  ```

## Git Release

- [ ] Commit all changes
  ```bash
  git add .
  git commit -m "Release v1.0.0"
  ```

- [ ] Create and push tag
  ```bash
  git tag -a v1.0.0 -m "Release v1.0.0"
  git push origin main
  git push origin v1.0.0
  ```

- [ ] Create GitHub Release
  1. Go to https://github.com/eshanks16/brix/releases/new
  2. Select tag: `v1.0.0`
  3. Title: `Brix Pizza v1.0.0 🍕`
  4. Copy content from `.github/release-notes-v1.0.0.md`
  5. Check "Set as latest release"
  6. Publish release

## Kubernetes Testing

- [ ] Test Kubernetes deployment with v1.0.0
  ```bash
  # Clean deploy
  kubectl delete namespace brix
  kubectl apply -f deployment/k8s/brix/
  kubectl apply -f deployment/k8s/mysql/

  # Wait for pods
  kubectl wait --for=condition=ready pod -n brix -l app=brix-pizza --timeout=300s

  # Check logs
  kubectl logs -n brix -l app=brix-pizza

  # Test the app
  kubectl port-forward -n brix svc/brix-pizza 8080:80
  # Open http://localhost:8080 and test registration/login/ordering
  ```

- [ ] Verify image is pulled from Docker Hub (not built locally)
  ```bash
  kubectl describe pod -n brix -l app=brix-pizza | grep "Image:"
  # Should show: eshanks16/brix-pizza:v1.0.0
  ```

## Post-Release

- [ ] Update Docker Hub repository description
  - Add link to GitHub repo
  - Add quick start instructions
  - Add supported tags

- [ ] Announce release (optional)
  - Social media
  - Blog post
  - Email list

- [ ] Monitor for issues
  - Check GitHub issues
  - Monitor Docker Hub pulls
  - Watch for community feedback

## Verification

Final checks:

- [ ] README.md shows correct version and deployment instructions
- [ ] CHANGELOG.md is updated
- [ ] GitHub release is published
- [ ] Docker Hub has v1.0.0 and latest tags
- [ ] Kubernetes deployment works end-to-end
- [ ] All tests pass

## Rollback Plan

If issues are found:

1. Delete GitHub release
2. Delete git tag: `git tag -d v1.0.0 && git push origin :refs/tags/v1.0.0`
3. Fix issues
4. Restart release process
