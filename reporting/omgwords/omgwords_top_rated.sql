WITH ratings_data AS
(SELECT
  users.id,
  users.username,
  profiles.ratings -> 'Data' AS data
FROM profiles LEFT JOIN users ON profiles.user_id = users.id
WHERE ratings -> 'Data' != 'null'),
csw19_regular_ratings AS
(SELECT
  id,
  username,
  data -> 'CSW19.classic.regular'->>'r' AS r,
  data -> 'CSW19.classic.regular'->>'rd' AS rd
FROM ratings_data),
csw19_rapid_ratings AS
(SELECT
  id,
  username,
  data -> 'CSW19.classic.rapid'->>'r' AS r,
  data -> 'CSW19.classic.rapid'->>'rd' AS rd
FROM ratings_data),
csw19_blitz_ratings AS
(SELECT
  id,
  username,
  data -> 'CSW19.classic.blitz'->>'r' AS r,
  data -> 'CSW19.classic.blitz'->>'rd' AS rd
FROM ratings_data),
csw19_ultrablitz_ratings AS
(SELECT
  id,
  username,
  data -> 'CSW19.classic.ultrablitz'->>'r' AS r,
  data -> 'CSW19.classic.ultrablitz'->>'rd' AS rd
FROM ratings_data),
nwl18_regular_ratings AS
(SELECT
  id,
  username,
  data -> 'NWL18.classic.regular'->>'r' AS r,
  data -> 'NWL18.classic.regular'->>'rd' AS rd
FROM ratings_data),
nwl18_rapid_ratings AS
(SELECT
  id,
  username,
  data -> 'NWL18.classic.rapid'->>'r' AS r,
  data -> 'NWL18.classic.rapid'->>'rd' AS rd
FROM ratings_data),
nwl18_blitz_ratings AS
(SELECT
  id,
  username,
  data -> 'NWL18.classic.blitz'->>'r' AS r,
  data -> 'NWL18.classic.blitz'->>'rd' AS rd
FROM ratings_data),
nwl18_ultrablitz_ratings AS
(SELECT
  id,
  username,
  data -> 'NWL18.classic.ultrablitz'->>'r' AS r,
  data -> 'NWL18.classic.ultrablitz'->>'rd' AS rd
FROM ratings_data)

SELECT
  username,
  r,
  rd
FROM csw19_regular_ratings
WHERE r IS NOT NULL
AND rd::FLOAT < 100.0
ORDER BY r::FLOAT DESC
LIMIT 10

