CREATE TABLE IF NOT EXISTS picks (
  user_id VARCHAR(25) NOT NULL,
  event_id VARCHAR(25) NOT NULL,
  picks TEXT[] NOT NULL,
  score SMALLINT,
  PRIMARY KEY (user_id, event_id)
);

