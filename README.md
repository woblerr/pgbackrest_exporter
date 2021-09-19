# pgBackRest Exporter

[![Actions Status](https://github.com/woblerr/pgbackrest_exporter/workflows/build/badge.svg)](https://github.com/woblerr/pgbackrest_exporter/actions)
[![Coverage Status](https://coveralls.io/repos/github/woblerr/pgbackrest_exporter/badge.svg?branch=master)](https://coveralls.io/github/woblerr/pgbackrest_exporter?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/woblerr/pgbackrest_exporter)](https://goreportcard.com/report/github.com/woblerr/pgbackrest_exporter)

Prometheus exporter for [pgBackRest](https://pgbackrest.org/).

The metrics are collected based on result of `pgbackrest info --output json` command. By default, the metrics are collected for all stanzas received by command. You can specify stanzas to collect metrics. You need to run exporter on the same host where pgBackRest was installed or inside Docker.

All metrics are collected for `pgBackRest >= v2.32`.
For earlier versions, some metrics may not be collected or have insignificant label values.

For example, the `pgbackrest_repo_status` metric will be absent for `pgBackRest <= v2.31`.
And for other metrics lable will be `repo_key="0"`.

## Collected metrics

The metrics provided by the client.

* `pgbackrest_stanza_status` - current stanza status.

    Labels: stanza.

* `pgbackrest_repo_status` - current repository status.

    Labels: cipher, repo_key, stanza.

    Values description for stanza and repo statuses:
    - `0`: ok,
    - `1`: missing stanza path,
    - `2`: no valid backups,
    - `3`: missing stanza data,
    - `4`: different across repos,
    - `5`: database mismatch across repos,
    - `6`: requested backup not found,
    - `99`: other.

* `pgbackrest_backup_info` - backup info.

    Labels: backrest_ver, backup_name, backup_type, database_id, pg_version, prior, repo_key, stanza, wal_start, wal_stop.

    Values description:
     - `1` - info about backup is exist.

* `pgbackrest_backup_duration_seconds` - backup duration in seconds.

    Labels: backup_name, backup_type, database_id, repo_key, stanza, start_time, stop_time.

* `pgbackrest_backup_size_bytes` - full uncompressed size of the database.

    Labels: backup_name, backup_type, database_id, repo_key, stanza.

* `pgbackrest_backup_delta_bytes` - amount of data in the database to actually backup.

    Labels: backup_name, backup_type, database_id, repo_key, stanza.

* `pgbackrest_backup_repo_size_bytes` - full compressed files size to restore the database from backup.

    Labels: backup_name, backup_type, database_id, repo_key, stanza.

* `pgbackrest_backup_repo_delta_bytes` - compressed files size in backup.

    Labels: backup_name, backup_type, database_id, repo_key, stanza.

* `pgbackrest_backup_full_since_last_completion_seconds` - seconds since the last completed full backup.

    Labels: stanza.

* `pgbackrest_backup_diff_since_last_completion_seconds` - seconds since the last completed full or differential backup.

    Labels: stanza.

* `pgbackrest_backup_incr_since_last_completion_seconds` - seconds since the last completed full, differential or incremental backup.

    Labels: stanza.

* `pgbackrest_wal_archive_status` - current WAL archive status.

    Labels: database_id, pg_version, repo_key, stanza, wal_max, wal_min.

    Values description:
    - `0` - any one of WALMin and WALMax have empty value, there is no correct information about WAL archiving,
    - `1` - both WALMin and WALMax have no empty values, there is correct information about WAL archiving.

## Getting Started
### Building and running

```bash
git clone https://github.com/woblerr/pgbackrest_exporter.git
cd pgbackrest_exporter
make build
./pgbackrest_exporter <flags>
```

Available configuration flags:

```bash
./pgbackrest_exporter --help
usage: pgbackrest_exporter [<flags>]

Flags:
  --help                      Show context-sensitive help (also try --help-long and --help-man).
  --prom.port="9854"          Port for prometheus metrics to listen on.
  --prom.endpoint="/metrics"  Endpoint used for metrics.
  --collect.interval=600      Collecting metrics interval in seconds.
  --backrest.config=""        Full path to pgBackRest configuration file.
  --backrest.config-include-path=""  
                              Full path to additional pgBackRest configuration files.
  --backrest.stanza-include="" ...  
                              Specific stanza for collecting metrics. Can be specified several times.
  --backrest.stanza-exclude="" ...  
                              Specific stanza to exclude from collecting metrics. Can be specified several times.
  --verbose.info              Enable additional metrics labels.
```

#### Additional description of flags.

Custom `config` and/or custom `config-include-path` for `pgbackrest` command can be specified via `--backrest.config` and `--backrest.config-include-path` flags. Full paths must be specified.<br>
For example, `--backrest.config=/tmp/pgbackrest.conf` and/or `--backrest.config-include-path=/tmp/pgbackrest/conf.d`.

Custom `stanza` for collecting metrics can be specified via `--backrest.stanza-include` flag. You can specify several stanzas.<br>
For example, `--backrest.stanza-include=demo1 --backrest.stanza-include=demo2`.<br>
For this case, metrics will be collected only for `demo1` and `demo2` stanzas.

Custom `stanza` to exclude from collecting metrics can be specified via `--backrest.stanza-exclude` flag. You can specify several stanzas.<br>
For example, `--backrest.stanza-exclude=demo1 --backrest.stanza-exclude=demo2`.<br>
For this case, metrics **will not be collected** for `demo1` and `demo2` stanzas.<br>
If the same stanza is specified for include and exclude flags, then metrics for this stanza will not be collected. 
The flag `--backrest.stanza-exclude` has a higher priority.<br>
For example, `--backrest.stanza-include=demo1 --backrest.stanza-exclude=demo1`.<br>
For this case, metrics **will not be collected** for `demo1` stanza.

When flag `--verbose.info` is specified - WALMin and WALMax are added as metric labels.<br>
This creates new different time series on each WAL archiving.

### Building and running docker
By default, pgBackRest version is `2.35`. Another version can be specified via arguments.
For base image used [docker-pgbackrest](https://github.com/woblerr/docker-pgbackrest) image.

```bash
make docker
```

or for specific pgBackRest version

```bash
docker build -f Dockerfile --build-arg BACKREST_VERSION=2.34 -t pgbackrest_exporter .
```

Environment variables supported by this image:
* all environment variables from [docker-pgbackrest](https://github.com/woblerr/docker-pgbackrest#docker-pgbackrest)  image;
* `EXPORTER_ENDPOINT` - metrics endpoint, default `/metrics`;
* `EXPORTER_PORT` - port for prometheus metrics to listen on, default `9854`;
* `STANZA_INCLUDE` - specific stanza for collecting metrics, default `""`;
* `STANZA_EXCLUDE` - specific stanza to exclude from collecting metrics, default `""`;
* `COLLECT_INTERVAL` - collecting metrics interval in seconds, default `600`;

You will need to mount the necessary directories or files inside the container.

Simple run:

```bash
docker run -d \
    --name pgbackrest_exporter \
    -p 9854:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \ 
    pgbackrest_exporter
```

With some enviroment variables:

```bash
docker run -d \
    --name pgbackrest_exporter \
    -e BACKREST_USER=postgres \
    -e BACKREST_UID=1001 \
    -e BACKREST_GROUP=postgres \
    -e BACKREST_GID=1001 \
    -e TZ=America/Chicago \
    -e COLLECT_INTERVAL=60 \
    -p 9854:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \
    pgbackrest_exporter
```

For specific stanza:

```bash
docker run -d \
    --name pgbackrest_exporter \
    -e STANZA_INCLUDE=demo \
    -p 9854:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \
    pgbackrest_exporter
```

If you want to specify several stanzas for collecting metrics, 
you can run containers on different ports:

```bash
docker run -d \
    --name pgbackrest_exporter_demo1 \
    -e STANZA_INCLUDE=demo1 \
    -p 9854:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \
    pgbackrest_exporter

docker run -d \
    --name pgbackrest_exporter_demo2 \
    -e STANZA_INCLUDE=demo2 \
    -p 9855:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \
    pgbackrest_exporter
```

To exclude specific stanza:

```bash
docker run -d \
    --name pgbackrest_exporter \
    -e STANZA_EXCLUDE=demo \
    -p 9854:9854 \
    -v  /etc/pgbackrest/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf \
    pgbackrest_exporter
```
### Running as systemd service

* Register `pgbackrest_exporter` (already builded, if not - exec `make build` before) as a systemd service:

```bash
 make make prepare-service
```

Validate prepared file `pgbackrest_exporter.service` and run:

```bash
sudo make install-service
```

* View service logs:

```bash
journalctl -u pgbackrest_exporter.service
```

* Delete systemd service:

```bash
sudo make remove-service
```

---
Manual register systemd service:

```bash
cp pgbackrest_exporter.service.template pgbackrest_exporter.service
```

In file `pgbackrest_exporter.service` replace `{PATH_TO_FILE}` to full path to `pgbackrest_exporter`.

```bash
sudo cp pgbackrest_exporter.service /etc/systemd/system/pgbackrest_exporter.service
sudo systemctl daemon-reload
sudo systemctl enable pgbackrest_exporter.service
sudo systemctl restart pgbackrest_exporter.service
systemctl -l status pgbackrest_exporter.service
```
