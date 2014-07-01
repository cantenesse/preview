package common

/*
CREATE DATABASE preview;
USE preview;
CREATE TABLE IF NOT EXISTS generated_assets (id varchar(80), source varchar(255), status varchar(80), template_id varchar(80), message blob, PRIMARY KEY (id));
CREATE TABLE IF NOT EXISTS active_generated_assets (id varchar(80) PRIMARY KEY);
CREATE TABLE IF NOT EXISTS waiting_generated_assets (id varchar(80), source varchar(80), template varchar(80), PRIMARY KEY(template, id, source));
CREATE TABLE IF NOT EXISTS source_assets (id varchar(80), type varchar(80), message blob, PRIMARY KEY (id, type));

TRUNCATE source_assets;
TRUNCATE generated_assets;
TRUNCATE active_generated_assets;
TRUNCATE waiting_generated_assets;

*/
