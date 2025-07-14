package sqlx

const (
	initTableSQL = `
		CREATE TABLE IF NOT EXISTS outbox
		(
			code            uuid               PRIMARY KEY,
			data            jsonb              NOT NULL,
			source		    TEXT               NOT NULL,
			created_at      timestamptz        NOT NULL DEFAULT now(),
			updated_at      timestamptz 	   NOT NULL DEFAULT now(),
			status          TEXT 			   NOT NULL DEFAULT 'new',
			failed_attempts INT 			   NOT NULL DEFAULT 0
		);
		
		CREATE INDEX IF NOT EXISTS outbox_status_idx ON outbox (status, created_at ASC);
`

	insertDataSQL = `
		INSERT INTO outbox (code, data, source) 
		VALUES ($1, $2, $3);
`

	updateDataSQL = `
		UPDATE
			outbox AS o
		SET
		    status = $1,
		    updated_at = now()
`

	selectDataSQL = updateDataSQL + `
		WHERE
			o.status = 'new'
			AND o.failed_attempts < $2
		RETURNING 
			o.code, 
			o.data,
			o.source;
`

	updateProcessedDataSQL = updateDataSQL + `
		WHERE
			o.status = 'processing'
			AND o.code IN (%s);
`

	updateLockedDataSQL = updateDataSQL + `
		WHERE
			o.status = 'processing'
		    AND o.updated_at < $2;
`

	updateFailedDataSQL = updateDataSQL + `
		, failed_attempts = failed_attempts + 1
		WHERE 
			o.status = 'processing'
			AND o.code IN (%s);
`
	removeOldDataSQL = `
		DELETE FROM
		    outbox AS o
		WHERE
			o.status = 'sent'
		    AND o.updated_at < $1;
		
`

	setFailedDataSQL = updateDataSQL + `
		WHERE 
			o.status = 'new'
			AND o.failed_attempts >= $2;
`
)
