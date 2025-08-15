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
    github_token: ${{ secrets.GITHUB_TOKEN }}
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

## Requirements
- `GITHUB_REPOSITORY` must reflect the repo to install (provided by Actions).

## Ploi API endpoints
- Create site: `POST /api/servers/{server}/sites` ([docs](https://developers.ploi.io/32-sites/91-create-site))
- Install repository: `POST /api/servers/{server}/sites/{site}/repository` ([docs](https://developers.ploi.io/40-repositories/124-install-repository))
- Create certificate: `POST /api/servers/{server}/sites/{site}/certificates` ([docs](https://developers.ploi.io/43-certificates/134-create-certificate))
- Update deploy script: `PUT /api/servers/{server}/sites/{site}/deploy-script` ([docs](https://developers.ploi.io/39-deployments/122-update-deploy-script))
- Update env: `PUT /api/servers/{server}/sites/{site}/env` ([docs](https://developers.ploi.io/41-environment/127-update-env-from-site))
- Create database: `POST /api/servers/{server}/databases` ([docs](https://developers.ploi.io/33-databases/96-create-database))
- Create database user: `POST /api/servers/{server}/database-users` ([docs](https://developers.ploi.io/523-database-users/1410-create-database-user))
- Create daemons: `POST /api/servers/{server}/sites/{site}/daemons`
