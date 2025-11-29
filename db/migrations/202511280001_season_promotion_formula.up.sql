-- Add promotion_formula column to league_seasons
-- Values correspond to PromotionFormula enum in proto:
-- 0 = PROMO_N_DIV_6 (ceil(N/6)) - default
-- 1 = PROMO_N_PLUS_1_DIV_5 (ceil((N+1)/5))
-- 2 = PROMO_N_DIV_5 (ceil(N/5))
ALTER TABLE league_seasons ADD COLUMN promotion_formula INT NOT NULL DEFAULT 0;
