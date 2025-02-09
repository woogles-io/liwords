BEGIN;

CREATE TABLE badges(
    id SMALLSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL, -- e.g. Chihuahua
    description TEXT NOT NULL
);

CREATE TABLE user_badges(
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    badge_id INT REFERENCES badges(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, badge_id)
);

INSERT INTO badges (code, description)
VALUES
    ('Pug', '$1 Pug tier contributor during the 2020 Woogles Kickstarter'),
    ('Greyhound', '$10 Greyhound tier contributor during the 2020 Woogles Kickstarter'),
    ('Beagle', '$25 Beagle tier contributor during the 2020 Woogles Kickstarter'),
    ('Shar-Pei', '$60 Shar-Pei tier contributor during the 2020 Woogles Kickstarter'),
    ('Afghan', '$120 Afghan tier contributor during the 2020 Woogles Kickstarter'),
    ('Corgi', '$250 Corgi tier contributor during the 2020 Woogles Kickstarter'),
    ('Dachshund', '$250 Dachshund tier contributor during the 2020 Woogles Kickstarter'),
    ('Collie', '$500 Collie tier contributor during the 2020 Woogles Kickstarter'),
    ('Basset Hound', '$1000 Basset Hound tier contributor during the 2020 Woogles Kickstarter'),
    ('Woogles', '$2500 Woogles tier contributor during the 2020 Woogles Kickstarter'),
    ('Chihuahua', 'Chihuahua tier subscriber on Patreon'),
    ('Dalmatian', 'Dalmatian tier subscriber on Patreon'),
    ('Golden Retriever', 'Golden Retriever tier subscriber on Patreon'),
    ('Cevapcici', 'Played CEVAPCICI in only a semi-contrived fashion. See https://woogles.io/game/87L8WVUb'),
    ('Most Social', 'Winner of the Most Social Dog award during the early Launch Parties');
COMMIT;