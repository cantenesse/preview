package common

/*
CREATE KEYSPACE preview
  WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 3 };
USE preview;
CREATE TABLE IF NOT EXISTS generated_assets (id timeuuid, source varchar, status varchar, template_id varchar, message blob, PRIMARY KEY (id));
CREATE TABLE IF NOT EXISTS active_generated_assets (id timeuuid PRIMARY KEY);
CREATE TABLE IF NOT EXISTS waiting_generated_assets (id timeuuid, source varchar, template varchar, PRIMARY KEY(template, id, source));
CREATE INDEX IF NOT EXISTS ON generated_assets (source);
CREATE INDEX IF NOT EXISTS ON generated_assets (status);
CREATE INDEX IF NOT EXISTS ON generated_assets (template_id);
CREATE TABLE IF NOT EXISTS source_assets (id varchar, type varchar, message blob, PRIMARY KEY (id, type));
CREATE INDEX IF NOT EXISTS ON source_assets (type);

TRUNCATE source_assets;
TRUNCATE generated_assets;
TRUNCATE active_generated_assets;
TRUNCATE waiting_generated_assets;

*/
