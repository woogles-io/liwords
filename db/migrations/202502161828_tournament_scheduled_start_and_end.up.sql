ALTER TABLE  tournaments
ADD COLUMN scheduled_start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT '1970-01-01 00:00:00+00',
ADD COLUMN scheduled_end_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT '1970-01-01 00:00:00+00';

CREATE INDEX idx_tournaments_scheduled_start_time ON public.tournaments(scheduled_start_time);
CREATE INDEX idx_tournaments_scheduled_end_time ON public.tournaments(scheduled_end_time);