-- Leagues table
CREATE TABLE leagues (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    slug TEXT UNIQUE NOT NULL,
    settings JSONB NOT NULL,
    current_season_id UUID,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by BIGINT
);

-- Seasons table
CREATE TABLE league_seasons (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    league_id UUID NOT NULL REFERENCES leagues(uuid) ON DELETE CASCADE,
    season_number INT NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    actual_end_date TIMESTAMP WITH TIME ZONE,
    status INTEGER NOT NULL, -- SeasonStatus proto enum: 0=SCHEDULED, 1=ACTIVE, 2=COMPLETED, 3=CANCELLED, 4=REGISTRATION_OPEN
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(league_id, season_number)
);

-- Divisions table
CREATE TABLE league_divisions (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    season_id UUID NOT NULL REFERENCES league_seasons(uuid) ON DELETE CASCADE,
    division_number INT NOT NULL,
    division_name TEXT,
    player_count INT,
    is_complete BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(season_id, division_number)
);

-- Registrations table
CREATE TABLE league_registrations (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    season_id UUID NOT NULL REFERENCES league_seasons(uuid) ON DELETE CASCADE,
    division_id UUID REFERENCES league_divisions(uuid) ON DELETE SET NULL,
    registration_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    firsts_count INT,
    status TEXT DEFAULT 'ACTIVE',
    placement_status INTEGER, -- StandingResult proto enum: 0=NONE, 1=PROMOTED, 2=RELEGATED, 3=STAYED, 4=CHAMPION
    previous_division_rank INT,
    seasons_away INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, season_id)
);

-- Standings cache table
CREATE TABLE league_standings (
    id BIGSERIAL PRIMARY KEY,
    division_id UUID NOT NULL REFERENCES league_divisions(uuid) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rank INT,
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    draws INT DEFAULT 0,
    spread INT DEFAULT 0,
    games_played INT DEFAULT 0,
    games_remaining INT,
    result INTEGER, -- StandingResult proto enum: 0=NONE, 1=PROMOTED, 2=RELEGATED, 3=STAYED, 4=CHAMPION
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(division_id, user_id)
);

-- Indexes for efficient queries
CREATE INDEX idx_league_seasons_league_id ON league_seasons(league_id);
CREATE INDEX idx_league_seasons_status ON league_seasons(status);
CREATE INDEX idx_league_divisions_season_id ON league_divisions(season_id);
CREATE INDEX idx_league_registrations_user_id ON league_registrations(user_id);
CREATE INDEX idx_league_registrations_season_id ON league_registrations(season_id);
CREATE INDEX idx_league_registrations_division_id ON league_registrations(division_id);
CREATE INDEX idx_league_standings_division_id ON league_standings(division_id);
CREATE INDEX idx_league_standings_user_id ON league_standings(user_id);
CREATE INDEX idx_leagues_slug ON leagues(slug);

-- Add league metadata to games table
ALTER TABLE games ADD COLUMN league_id UUID;
ALTER TABLE games ADD COLUMN season_id UUID;
ALTER TABLE games ADD COLUMN league_division_id UUID;

CREATE INDEX idx_games_league_id ON games(league_id) WHERE league_id IS NOT NULL;
CREATE INDEX idx_games_season_id ON games(season_id) WHERE season_id IS NOT NULL;
CREATE INDEX idx_games_league_division_id ON games(league_division_id) WHERE league_division_id IS NOT NULL;
