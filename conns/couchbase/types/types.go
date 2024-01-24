package types

import "github.com/couchbase/gocb/v2"

type (
	CollectionTx struct {
		Ctx        *gocb.TransactionAttemptContext
		Collection *gocb.Collection
		Bucket     *gocb.Bucket
	}
)

func (c *CollectionTx) Query(statement string, options *gocb.TransactionQueryOptions) (*gocb.TransactionQueryResult, error) {
	return c.Ctx.Query(statement, options)
}

func (c *CollectionTx) Internal() *gocb.InternalTransactionAttemptContext {
	return c.Ctx.Internal()
}

func (c *CollectionTx) Get(id string) (*gocb.TransactionGetResult, error) {
	return c.Ctx.Get(c.Collection, id)
}

func (c *CollectionTx) Replace(doc *gocb.TransactionGetResult, value interface{}) (*gocb.TransactionGetResult, error) {
	return c.Ctx.Replace(doc, value)
}
func (c *CollectionTx) Insert(id string, value interface{}) (*gocb.TransactionGetResult, error) {
	return c.Ctx.Insert(c.Collection, id, value)
}

func (c *CollectionTx) Remove(doc *gocb.TransactionGetResult) error {
	return c.Ctx.Remove(doc)
}

func (c *CollectionTx) GetScope(scopeName string) *gocb.Scope {
	return c.Bucket.Scope(scopeName)
}
