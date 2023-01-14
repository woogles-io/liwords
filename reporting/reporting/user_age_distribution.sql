SELECT
  -- some signups prior to 6/19/21 do not have birth_date specified
  CASE 
    WHEN (birth_date != '' AND birth_date > '1900-01-01') THEN ROUND((NOW()::date-birth_date::date)/365.25,0)
	ELSE NULL END AS age,
  COUNT(*)
FROM public.profiles
GROUP BY 1
ORDER BY 1 ASC