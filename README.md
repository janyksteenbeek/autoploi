# autoploi (GitHub Action)

Deploy and manage Ploi sites from GitHub Actions.

- deploy: create site, install repository, request Let's Encrypt, update deploy script, optionally create DB+user and set `DATABASE_URL`, apply .env lines, create daemons, and comment the PR with the URL.
- find-site-by-domain: look up a site ID by domain on a given server.
- delete-site: delete a site by ID.

## Usage (deploy)

```yaml
- name: Autoploi (deploy)
  uses: janyksteenbeek/autoploi@v1
  with:
    action: deploy
    ploi_token: ${{ secrets.PLOI_TOKEN }}
    server_id: "12345"
    domain: example.com
    branch: ${{ github.ref_name }}
    deploy_script: |
      php artisan down || true
      composer install --no-interaction --prefer-dist --optimize-autoloader
      php artisan migrate --force
      php artisan up
    environment: |
      APP_ENV=production
      APP_DEBUG=false
    daemons: |
      - command: php artisan queue:work --queue=default --sleep=1 --tries=3
        path: /home/ploi/example.com/current
      - php artisan schedule:work
    create_database: "true"
    database_engine: mysql
    database_host: 127.0.0.1
    database_port: "3306"
```

Outputs:
- `site_id`: created site ID
- `url`: https://example.com

## Usage (find-site-by-domain)

```yaml
- name: Find site by domain
  id: find
  uses: janyksteenbeek/autoploi@v1
  with:
    action: find-site-by-domain
    ploi_token: ${{ secrets.PLOI_TOKEN }}
    server_id: "12345"
    domain: example.com
```

Outputs:
- `site_id`: found site ID

## Usage (delete-site)

```yaml
- name: Delete site
  uses: janyksteenbeek/autoploi@v1
  with:
    action: delete-site
    ploi_token: ${{ secrets.PLOI_TOKEN }}
    server_id: "12345"
    site_id: ${{ steps.find.outputs.site_id }}
```

## Example: use find-site-by-domain output in next step

```yaml
yaml
jobs:
  manage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Find site by domain
        id: find
        uses: janyksteenbeek/autoploi@v1
        with:
          action: find-site-by-domain
          ploi_token: ${{ secrets.PLOI_TOKEN }}
          server_id: "12345"
          domain: example.com

      - name: Delete found site
        uses: janyksteenbeek/autoploi@v1
        with:
          action: delete-site
          ploi_token: ${{ secrets.PLOI_TOKEN }}
          server_id: "12345"
          site_id: ${{ steps.find.outputs.site_id }}
```

Optional: allow missing site without failing the job

```yaml
yaml
- name: Find site by domain (allow missing)
  id: find
  continue-on-error: true
  uses: janyksteenbeek/autoploi@v1
  with:
    action: find-site-by-domain
    ploi_token: ${{ secrets.PLOI_TOKEN }}
    server_id: "12345"
    domain: example.com

- name: Delete only if found
  if: steps.find.outputs.site_id != ''
  uses: janyksteenbeek/autoploi@v1
  with:
    action: delete-site
    ploi_token: ${{ secrets.PLOI_TOKEN }}
    server_id: "12345"
    site_id: ${{ steps.find.outputs.site_id }}
```