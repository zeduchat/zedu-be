DELETE FROM organisation_integrations a USING (
    SELECT MIN(ctid) as ctid, pre_shared_key
    FROM organisation_integrations 
    WHERE pre_shared_key IS NOT NULL
    GROUP BY pre_shared_key 
    HAVING COUNT(*) > 1
) b
WHERE a.pre_shared_key = b.pre_shared_key 
AND a.ctid <> b.ctid;

CREATE UNIQUE INDEX IF NOT EXISTS idx_organisation_integrations_pre_shared_key
ON organisation_integrations (pre_shared_key);