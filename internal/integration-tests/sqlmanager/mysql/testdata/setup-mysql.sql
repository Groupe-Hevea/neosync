-- MySQL-only fixtures: expression indexes are not supported by MariaDB (MDEV-35853)

USE sqlmanagermysql4;

-- Create a table with some columns
CREATE TABLE test_mixed_index (
    id INT PRIMARY KEY,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    birth_date DATE
);

-- Create a composite index that uses both regular columns and expressions
CREATE INDEX idx_mixed ON test_mixed_index
    (first_name, (UPPER(last_name)), birth_date, (YEAR(birth_date)));
