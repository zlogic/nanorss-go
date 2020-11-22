package datadb

import (
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
)

const migrationCreateSchema = `
-- Database schema version
CREATE TABLE schema_version (
	version integer NOT NULL
);

-- Feeds
CREATE TABLE feeds (
	id  SERIAL NOT NULL PRIMARY KEY,
	url TEXT NOT NULL UNIQUE,

	last_success       TIMESTAMP,
	last_failure       TIMESTAMP,
	last_failure_error TEXT
);

-- Feed items
CREATE TABLE feeditems (
	feed_id  INT NOT NULL,
	guid     TEXT NOT NULL,
	title    TEXT NOT NULL,
	url      TEXT NOT NULL,
	date     TIMESTAMP NOT NULL,
	contents TEXT NOT NULL,
	updated  TIMESTAMP NOT NULL,

	last_seen TIMESTAMP,

	PRIMARY KEY(feed_id, guid),
	FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
);

-- Page monitor pages
CREATE TABLE pagemonitors (
	id      SERIAL NOT NULL PRIMARY KEY,
	url     TEXT NOT NULL,
	match   TEXT NOT NULL,
	replace TEXT NOT NULL,

	contents TEXT NOT NULL,
	delta    TEXT NOT NULL,
	updated  TIMESTAMP NOT NULL,

	last_success       TIMESTAMP,
	last_failure       TIMESTAMP,
	last_failure_error TEXT,

	UNIQUE(url, match, replace)
);

-- User details
CREATE TABLE users (
	id       SERIAL NOT NULL PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,

	password    TEXT,
	opml        TEXT,
	pagemonitor TEXT
);

-- User's associations
CREATE TABLE user_feeds (
	user_id INT NOT NULL,
	feed_id INT NOT NULL,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, feed_id)
);

CREATE TABLE user_pagemonitors (
	user_id         INT NOT NULL,
	pagemonitor_id  INT NOT NULL,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(pagemonitor_id) REFERENCES pagemonitors(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, pagemonitor_id)
);

CREATE TABLE user_read_feeditems (
	user_id     INT NOT NULL,
	feeditem_id INT NOT NULL,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(feeditem_id) REFERENCES feeds(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, feeditem_id)
);

CREATE TABLE user_read_pagemonitors (
	user_id        INT NOT NULL,
	pagemonitor_id INT NOT NULL,

	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY(pagemonitor_id) REFERENCES pagemonitors(id) ON DELETE CASCADE,
	PRIMARY KEY(user_id, pagemonitor_id)
);

-- Server configuration
CREATE TABLE serverconfig (
	key   TEXT NOT NULL PRIMARY KEY,
	value TEXT NOT NULL
);

-- Populate tables with initial data
INSERT INTO schema_version(version) VALUES (1);
`

func applyMigrations(tx *sql.Tx) error {
	var exists int
	err := tx.QueryRow("SELECT 1 FROM information_schema.tables WHERE table_name=$1", "schema_version").Scan(&exists)

	var currentVersion int
	if err == sql.ErrNoRows {
		log.Info("Schema version table is missing, assuming empty database")
		currentVersion = 0
	} else if err != nil {
		return err
	} else {
		err = tx.QueryRow("SELECT version FROM schema_version").Scan(&currentVersion)
		if err != nil {
			return fmt.Errorf("failed to get current schema version: %w", err)
		}
	}

	if currentVersion == 1 {
		log.Info("Schema is up to date")
		return nil
	}

	_, err = tx.Exec(migrationCreateSchema)
	return err
}
