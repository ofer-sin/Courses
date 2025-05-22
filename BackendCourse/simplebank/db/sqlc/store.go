// Implement db transactions

package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	// Embedding the Querier interface from mockgen package to access all its methods
	// The Store interface embeds the Querier interface, which means it inherits all the methods defined in the Querier interface.
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
}

type SQLStore struct {
	*Queries // Embedding Queries struct to access all its methods (just like inheritance)
	db       *sql.DB
}

// NewStore creates a new Store instance with the provided database connection.
func NewStore(db *sql.DB) Store {
	return &SQLStore{
		Queries: New(db),
		db:      db,
	}
}

// execTx executes a function within a database transaction.
// It begins a transaction, executes the provided function, and commits or rolls back the transaction based on the function's result.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
// The function receives a Queries instance that is bound to the transaction, allowing it to perform database operations within the transaction context.
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx) // Create a new Queries instance with the transaction

	err = fn(q) // Execute the function with the Queries instance
	if err != nil {
		// If there was an error, rollback the transaction
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// TransferTxParams contains the parameters for the TransferTx function.
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult contains the result of the TransferTx function.
type TransferTxResult struct {
	Transfer    Transfers `json:"transfer"`
	FromAccount Accounts  `json:"from_account"`
	ToAccount   Accounts  `json:"to_account"`
	FromEntry   Entries   `json:"from_entry"`
	ToEntry     Entries   `json:"to_entry"`
}

// this is used to transfer the transaction name to the context
// var txKey = struct{}{}

// TransferTx performs a money transfer from one account to another.
// It creates a transfer record, updates the account balances, and creates entry records for both accounts.
// It uses a transaction to ensure atomicity, meaning that either all operations succeed or none do.
// The function returns a TransferTxResult containing the details of the transfer and the updated account balances.
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	// Start the transaction
	// Thw Queries instance is bound to the transaction
	// and will be used to perform all database operations within the transaction.
	// All the transaction operations are executed in the function passed to execTx.
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		// txName := ctx.Value(txKey)

		// Create a transfer record
		// fmt.Println(txName, "CreateTransfer")
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		// Create an entry record for the from account
		// fmt.Println(txName, "CreateEntry1")
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		// Create an entry record for the to account
		// fmt.Println(txName, "CreateEntry2")
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// to avoid deadlock, we need to update the account balance in the same order. We will always
		// update the account with the smaller ID first.
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMonney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMonney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}
		if err != nil {
			return err
		}
		// fmt.Println(txName, "UpdateAccount1")	}

		return nil
	})

	return result, err
}

func addMonney(ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Accounts, account2 Accounts, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:      accountID1,
		Ammount: amount1,
	})
	if err != nil {
		return // same as "return account1, account2, err"
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:      accountID2,
		Ammount: amount2,
	})

	return
}
