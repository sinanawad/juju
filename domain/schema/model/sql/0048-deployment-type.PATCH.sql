CREATE TABLE deployment_type (
    id INT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO deployment_type VALUES
(0, 'stateful'),
(1, 'stateless'),
(2, 'daemon');

ALTER TABLE application ADD COLUMN deployment_type_id INT NOT NULL DEFAULT 0
    REFERENCES deployment_type(id);
