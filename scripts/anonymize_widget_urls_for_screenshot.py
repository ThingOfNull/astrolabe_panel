#!/usr/bin/env python3
"""Temporary anonymize URLs in SQLite widget rows for screenshots.

Rewrites:

- type ``link``: ``config.url`` (http/https), ``probe.host`` when ``probe.type == tcp``.
- type ``search``: each ``engines[].url`` (keeps placeholders like ``{q}``).

Hosts become ``192.168.x.y``, scheme ``http``; path / query / fragment kept.

Stop the astrolabe process before running to avoid WAL conflicts.

Usage::

    anonymize_widget_urls_for_screenshot.py [--db PATH] [--start N]

If ``--db`` is omitted, uses ``~/.astrolabe_panel/astrolabe.db``. If that file is
missing, prints an error and exits (pass ``--db`` explicitly).

After replacing rows you are prompted to take screenshots; press Enter here to restore
from the backup copied before mutation.

Only link/search widgets are changed; datasource endpoints are untouched.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import shutil
import sqlite3
import sys
from pathlib import Path
from urllib.parse import urlparse, urlunparse


def parse_args(argv: list[str]) -> argparse.Namespace:
    p = argparse.ArgumentParser(description='Anonymize link/search widget URLs for screenshots.')
    p.add_argument(
        '--db',
        dest='db_path',
        metavar='PATH',
        default=None,
        help='SQLite DB path (omit only if ~/.astrolabe_panel/astrolabe.db exists)',
    )
    p.add_argument(
        '--start',
        type=int,
        default=1,
        metavar='N',
        help='Host counter maps to sequential 192.168.*.* starting here (default: 1 → 192.168.1.1)',
    )
    return p.parse_args(argv)


def resolve_db_path(cli_db: str | None) -> Path:
    if cli_db:
        return Path(cli_db).expanduser().resolve()

    fallback = (Path.home() / '.astrolabe_panel' / 'astrolabe.db').expanduser().resolve()
    if not fallback.is_file():
        print(
            'error: default database not found:',
            fallback,
            file=sys.stderr,
        )
        print(
            'Specify explicitly: {} --db /path/to/astrolabe.db'.format(Path(sys.argv[0]).name),
            file=sys.stderr,
        )
        sys.exit(1)
    return fallback


def backup_path_for(db: Path) -> Path:
    stamp = dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    return db.parent / f'{db.name}.anonymize_backup.{stamp}'


def host_for_sequence_index(seq: int) -> str:
    """seq starts at ``--start`` (default 1); maps to 192.168.1.1 .. 192.168.1.254, then 192.168.2.1 .."""
    if seq < 1:
        raise ValueError('sequence index must be >= 1')

    zero = seq - 1
    subnet = zero // 254
    fourth = zero % 254 + 1
    third = 1 + subnet
    return '192.168.%d.%d' % (third, fourth)


def anonymize_http_url(raw: str, seq: int) -> str:
    trimmed = raw.strip()
    parsed = urlparse(trimmed)

    hostname = host_for_sequence_index(seq)
    if parsed.port is not None and parsed.port not in (80, 443):
        netloc = '%s:%d' % (hostname, parsed.port)
    else:
        netloc = hostname

    path = parsed.path if parsed.path else '/'
    return urlunparse(
        (
            'http',
            netloc,
            path,
            parsed.params,
            parsed.query,
            parsed.fragment,
        ),
    )


def anonymize_tcp_host(raw: str, seq: int) -> str:
    s = raw.strip()
    if not s:
        return s
    hostname = host_for_sequence_index(seq)
    if ':' in s:
        _candidate, tail = s.rsplit(':', 1)
        if tail.isdigit():
            return '%s:%s' % (hostname, tail)
    return hostname


def mutate_widget_configs(conn: sqlite3.Connection, start_seq: int) -> tuple[int, int]:
    """Returns ``(widgets_updated, replacements_made)``."""
    cur = conn.execute(
        'SELECT id, type, config FROM widgets WHERE type IN (\'link\', \'search\') ORDER BY id ASC',
    )
    rows = cur.fetchall()

    seq = start_seq
    widgets_touched = 0
    fields = 0

    for wid, typ, conf in rows:
        if conf is None or conf == '':
            continue
        try:
            obj = json.loads(conf)
        except json.JSONDecodeError:
            continue

        dirty = False
        if typ == 'link':
            u = obj.get('url')
            if isinstance(u, str) and u.strip():
                low = u.lower().strip()
                if low.startswith(('http://', 'https://')):
                    obj['url'] = anonymize_http_url(u, seq)
                    seq += 1
                    fields += 1
                    dirty = True

            probe = obj.get('probe')
            if isinstance(probe, dict) and probe.get('type') == 'tcp':
                h = probe.get('host')
                if isinstance(h, str) and h.strip():
                    probe['host'] = anonymize_tcp_host(h, seq)
                    seq += 1
                    fields += 1
                    dirty = True

        elif typ == 'search':
            engines = obj.get('engines')
            if isinstance(engines, list):
                for eng in engines:
                    if not isinstance(eng, dict):
                        continue
                    eu = eng.get('url')
                    if isinstance(eu, str) and eu.strip():
                        elt = eu.lower().strip()
                        if elt.startswith(('http://', 'https://')):
                            eng['url'] = anonymize_http_url(eu, seq)
                            seq += 1
                            fields += 1
                            dirty = True

        if dirty:
            new_conf = json.dumps(obj, ensure_ascii=False, separators=(',', ':'))
            conn.execute('UPDATE widgets SET config = ? WHERE id = ?', (new_conf, wid))
            widgets_touched += 1

    return widgets_touched, fields


def main(argv: list[str]) -> None:
    args = parse_args(argv)
    if args.start < 1:
        print('error: --start must be >= 1', file=sys.stderr)
        sys.exit(1)

    db = resolve_db_path(args.db_path)
    if not db.is_file():
        print('error: database file does not exist:', db, file=sys.stderr)
        sys.exit(1)

    bak = backup_path_for(db)
    shutil.copy2(db, bak)

    conn = sqlite3.connect(str(db))
    widgets_touched, fields_count = mutate_widget_configs(conn, args.start)
    conn.commit()
    conn.close()

    print('backup written to:', bak)
    print('updated widgets:', widgets_touched, '  replaced url/host fields:', fields_count)
    print()
    print('>>> 请先停止 astrolabe，再启动并打开页面截屏（数据库已写入匿名 URL）。')
    print('>>> 截完图后回到此终端，按 Enter 将从备份还原数据库 …')
    try:
        input()
    except EOFError:
        print(
            'error: EOF on stdin; restore manually with e.g. cp',
            bak,
            db,
            file=sys.stderr,
        )
        sys.exit(2)

    shutil.copy2(bak, db)
    print('restored from backup:', bak)
    print('You may delete the backup file if it is no longer needed:', bak)


if __name__ == '__main__':
    main(sys.argv[1:])
