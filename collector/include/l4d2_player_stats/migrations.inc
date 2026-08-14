#if defined _l4d2_player_stats_migrations_included
	#endinput
#endif
#define _l4d2_player_stats_migrations_included

bool LPS_GetMigrationMetadata(int version, char[] fileName, int fileNameLength, char[] name, int nameLength)
{
	switch (version)
	{
		case 1:
		{
			strcopy(fileName, fileNameLength, "0001_initial.sql");
			strcopy(name, nameLength, "initial_schema");
			return true;
		}
		case 2:
		{
			strcopy(fileName, fileNameLength, "0002_car_alarms_triggered.sql");
			strcopy(name, nameLength, "car_alarms_triggered");
			return true;
		}
		case 3:
		{
			strcopy(fileName, fileNameLength, "0003_versus_objective_interactions.sql");
			strcopy(name, nameLength, "versus_objective_interactions");
			return true;
		}
		case 4:
		{
			strcopy(fileName, fileNameLength, "0004_analysis_foundation.sql");
			strcopy(name, nameLength, "analysis_foundation");
			return true;
		}
		case 5:
		{
			strcopy(fileName, fileNameLength, "0005_relationships_and_assists.sql");
			strcopy(name, nameLength, "relationships_and_assists");
			return true;
		}
	}
	return false;
}

bool LPS_BuildMigrationPath(int version, char[] path, int maxLength)
{
	if (g_LPSDriverName[0] == '\0')
	{
		return false;
	}

	char fileName[64];
	char migrationName[64];
	if (!LPS_GetMigrationMetadata(version, fileName, sizeof(fileName), migrationName, sizeof(migrationName)))
	{
		return false;
	}

	BuildPath(
		Path_SM,
		path,
		maxLength,
		"configs/l4d2_player_stats/migrations/%s/%s",
		g_LPSDriverName,
		fileName
	);
	return true;
}

bool LPS_ReadBootstrapStatement(char[] query, int maxLength, char[] error, int errorLength)
{
	query[0] = '\0';

	char path[PLATFORM_MAX_PATH];
	if (!LPS_BuildMigrationPath(1, path, sizeof(path)))
	{
		strcopy(error, errorLength, "database driver has no migration path");
		return false;
	}

	File file = OpenFile(path, "rt");
	if (file == null)
	{
		Format(error, errorLength, "migration file not found: %s", path);
		return false;
	}

	char line[LPS_MAX_MIGRATION_LINE];
	while (file.ReadLine(line, sizeof(line)))
	{
		TrimString(line);

		if (StrEqual(line, "-- statement-breakpoint", true))
		{
			break;
		}

		if (line[0] == '\0' || StrContains(line, "--", true) == 0)
		{
			continue;
		}

		if (!LPS_AppendMigrationLine(query, maxLength, line))
		{
			delete file;
			strcopy(error, errorLength, "bootstrap SQL statement exceeds the query buffer");
			return false;
		}
	}

	delete file;

	if (query[0] == '\0')
	{
		strcopy(error, errorLength, "migration bootstrap statement is empty");
		return false;
	}

	return true;
}

Transaction LPS_LoadMigration(int version, char[] error, int errorLength, int &statementCount)
{
	statementCount = 0;

	char path[PLATFORM_MAX_PATH];
	if (!LPS_BuildMigrationPath(version, path, sizeof(path)))
	{
		strcopy(error, errorLength, "database driver has no migration path");
		return null;
	}

	File file = OpenFile(path, "rt");
	if (file == null)
	{
		Format(error, errorLength, "migration file not found: %s", path);
		return null;
	}

	Transaction transaction = new Transaction();
	char query[LPS_MAX_QUERY_LENGTH];
	char line[LPS_MAX_MIGRATION_LINE];
	query[0] = '\0';

	while (file.ReadLine(line, sizeof(line)))
	{
		TrimString(line);

		if (StrEqual(line, "-- statement-breakpoint", true))
		{
			if (!LPS_AddMigrationStatement(transaction, query, statementCount))
			{
				delete file;
				delete transaction;
				strcopy(error, errorLength, "migration contains an empty statement");
				return null;
			}
			continue;
		}

		if (line[0] == '\0' || StrContains(line, "--", true) == 0)
		{
			continue;
		}

		if (!LPS_AppendMigrationLine(query, sizeof(query), line))
		{
			delete file;
			delete transaction;
			strcopy(error, errorLength, "a migration SQL statement exceeds the query buffer");
			return null;
		}
	}

	delete file;

	if (query[0] != '\0')
	{
		transaction.AddQuery(query);
		statementCount++;
	}

	if (statementCount < 1)
	{
		delete transaction;
		strcopy(error, errorLength, "migration contains no SQL statements");
		return null;
	}

	char migrationQuery[512];
	char fileName[64];
	char migrationName[64];
	if (!LPS_GetMigrationMetadata(version, fileName, sizeof(fileName), migrationName, sizeof(migrationName)))
	{
		delete transaction;
		strcopy(error, errorLength, "migration metadata is missing");
		return null;
	}
	Format(
		migrationQuery,
		sizeof(migrationQuery),
		"INSERT INTO lps_schema_migrations (version, name, applied_at) VALUES (%d, '%s', %d)",
		version,
		migrationName,
		GetTime()
	);
	transaction.AddQuery(migrationQuery);
	statementCount++;

	return transaction;
}

bool LPS_AppendMigrationLine(char[] query, int maxLength, const char[] line)
{
	int required = strlen(query) + strlen(line) + 2;
	if (required >= maxLength)
	{
		return false;
	}

	StrCat(query, maxLength, line);
	StrCat(query, maxLength, " ");
	return true;
}

bool LPS_AddMigrationStatement(Transaction transaction, char[] query, int &statementCount)
{
	if (query[0] == '\0')
	{
		return false;
	}

	transaction.AddQuery(query);
	statementCount++;
	query[0] = '\0';
	return true;
}
