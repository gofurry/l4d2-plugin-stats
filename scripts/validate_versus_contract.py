from __future__ import annotations

import json
import re
from dataclasses import dataclass
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parent.parent
CONTRACT_PATH = PROJECT_ROOT / "contracts" / "versus-schema-v1.json"
MIGRATION_ROOT = PROJECT_ROOT / "database" / "migrations"
DRIVERS = ("sqlite", "mysql", "pgsql")


@dataclass(frozen=True)
class TableDefinition:
    columns: tuple[str, ...]
    primary_key: tuple[str, ...]
    declarations: tuple[str, ...]


def split_table_items(body: str) -> list[str]:
    items: list[str] = []
    current: list[str] = []
    depth = 0
    for character in body:
        if character == "(":
            depth += 1
        elif character == ")":
            depth -= 1
        if character == "," and depth == 0:
            item = "".join(current).strip()
            if item:
                items.append(item)
            current = []
        else:
            current.append(character)
    item = "".join(current).strip()
    if item:
        items.append(item)
    return items


def normalize_declaration(item: str) -> str:
    return re.sub(r"\s+", " ", item.strip()).upper()


def parse_table(sql: str, table: str) -> TableDefinition:
    match = re.search(
        rf"CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+{re.escape(table)}\s*\((.*?)\)\s*;",
        sql,
        re.IGNORECASE | re.DOTALL,
    )
    if match is None:
        raise AssertionError(f"missing table {table}")

    columns: list[str] = []
    declarations: list[str] = []
    primary_key: list[str] = []
    for item in split_table_items(match.group(1)):
        normalized = normalize_declaration(item)
        if normalized.startswith("PRIMARY KEY"):
            key_match = re.search(r"PRIMARY KEY\s*\(([^)]+)\)", item, re.IGNORECASE)
            if key_match is None:
                raise AssertionError(f"could not parse primary key for {table}")
            primary_key = [part.strip().strip('`"') for part in key_match.group(1).split(",")]
            continue
        if normalized.startswith(("INDEX ", "KEY ", "UNIQUE ", "FOREIGN KEY", "CONSTRAINT ")):
            continue

        column_match = re.match(r"[`\"]?([A-Za-z_][A-Za-z0-9_]*)[`\"]?\s+(.+)", item, re.DOTALL)
        if column_match is None:
            raise AssertionError(f"could not parse {table} item: {item!r}")
        column = column_match.group(1)
        declaration = normalize_declaration(column_match.group(2))
        columns.append(column)
        declarations.append(declaration)
        if "PRIMARY KEY" in declaration:
            primary_key = [column]

    return TableDefinition(tuple(columns), tuple(primary_key), tuple(declarations))


def main() -> None:
    contract = json.loads(CONTRACT_PATH.read_text(encoding="utf-8"))
    if contract.get("contract_version") != 1 or contract.get("stats_version") != 1:
        raise AssertionError("unexpected Versus contract or stats version")

    expected_tables: dict[str, dict[str, list[str]]] = contract["tables"]
    parsed: dict[str, dict[str, TableDefinition]] = {}
    for driver in DRIVERS:
        migration = MIGRATION_ROOT / driver / "0001_initial.sql"
        sql = migration.read_text(encoding="utf-8")
        driver_tables: dict[str, TableDefinition] = {}
        actual_versus_tables = set(
            re.findall(
                r"CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+(lps_versus_[A-Za-z0-9_]+)",
                sql,
                re.IGNORECASE,
            )
        )
        if actual_versus_tables != set(expected_tables):
            raise AssertionError(
                f"{driver} Versus tables differ: "
                f"expected={sorted(expected_tables)} actual={sorted(actual_versus_tables)}"
            )

        for table, expected in expected_tables.items():
            definition = parse_table(sql, table)
            if list(definition.columns) != expected["columns"]:
                raise AssertionError(
                    f"{driver}.{table} columns differ:\n"
                    f"expected={expected['columns']}\nactual={list(definition.columns)}"
                )
            if list(definition.primary_key) != expected["primary_key"]:
                raise AssertionError(
                    f"{driver}.{table} primary key differs: "
                    f"expected={expected['primary_key']} actual={list(definition.primary_key)}"
                )
            driver_tables[table] = definition
        parsed[driver] = driver_tables

    baseline = parsed["sqlite"]
    for driver in ("mysql", "pgsql"):
        for table in expected_tables:
            if parsed[driver][table].declarations != baseline[table].declarations:
                raise AssertionError(
                    f"{driver}.{table} column declarations are not equivalent to sqlite"
                )

    print(
        f"Versus contract v1 validated across {len(DRIVERS)} drivers: "
        f"{len(expected_tables)} tables."
    )


if __name__ == "__main__":
    main()
