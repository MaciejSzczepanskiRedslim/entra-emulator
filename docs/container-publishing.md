# Publishing development container images

The optional `scripts/push-acr.sh` helper builds the existing Dockerfile for
`linux/amd64` and publishes it to an Azure Container Registry using your current
Azure CLI login. Docker must be running, and your account needs push access.

## Local configuration

Copy the example configuration:

```sh
cp .env.example .env
```

Set `ACR_NAME` in `.env` to your registry's resource name, without the
`.azurecr.io` suffix. The script reads its login-server address from Azure.
The `.env` file is excluded from Git and the Docker build context. You can also
provide `ACR_NAME` through your environment when no `.env` file is present.

## Publish a development image

```sh
./scripts/push-acr.sh
```

The image repository is `<login-server>/entra-emulator`. Each build publishes:

- `dev-<12-character Git commit>`: a tag identifying the source commit. Local
  uncommitted changes append `-dirty` to this tag.
- `dev`: a rolling tag for the newest development build.

Use a commit tag to pin a consumer to a particular build, or `dev` to test the
latest development image.
