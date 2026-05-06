//nolint:mnd
package seed

import (
	"context"
	"fmt"
	"time"

	fake "github.com/brianvoe/gofakeit/v7"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/test/factory"
	"github.com/ognev-dev/goplease/test/factory/random"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

// Users seeds the database with `count` random users.
//
// It generates unique usernames and emails, assigns randomized timestamps
// (created/updated/deleted), and inserts users concurrently using the factory.
func (s *Seed) Users(ctx context.Context, count int) (err error) {
	if count < 1 {
		return fmt.Errorf("seed users: %w", ErrInvalidCount)
	}

	uniqueUsername := func(from string) (string, error) {
		return factory.LookupUnique(ctx, s.db, "users", "username", from, func(s string) string {
			return s + "." + random.String(5)
		})
	}

	uniqueEmail := func(from string) (string, error) {
		return factory.LookupUnique(ctx, s.db, "users", "email", from, func(s string) string {
			return random.String(5) + "." + s
		})
	}

	var eg errgroup.Group

	for range count {
		eg.Go(func() error {
			createdAt := fake.DateRange(time.Now().AddDate(0, -12, 0), time.Now())
			// nil or after createdAt
			updatedAt := random.NilOrValue(fake.DateRange(createdAt.AddDate(0, -12, -25), createdAt), 75)
			// nil or after createdAt
			deletedAt := random.NilOrValue(fake.DateRange(createdAt.AddDate(0, -12, -25), createdAt), 75)

			u := ds.User{
				Username:       fake.Username(),
				Email:          fake.Email(),
				EmailConfirmed: random.Bool(),
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
				DeletedAt:      deletedAt,
			}

		createUser:
			_, err = s.factory.CreateUser(u)
			if column, ok := app.IsUniqueViolation(err); ok {
				switch column {
				case "username":
					u.Username, err = uniqueUsername(u.Username)
					if err != nil {
						return err
					}
				case "email":
					u.Email, err = uniqueEmail(u.Email)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("%w: column: %q", app.ErrUniqueViolation, column)
				}

				goto createUser
			}

			return err
		})
	}

	err = eg.Wait()
	if err != nil {
		return
	}

	// make sure one Book.PublicID is just "test",
	// so we can easily do manual tests
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.MinCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, "UPDATE users SET username='test', email='test', email_confirmed=true, deleted_at=NULL, password=$1 WHERE id=(SELECT id FROM users ORDER BY RANDOM() LIMIT 1)", string(passwordHash))
	if err != nil {
		return err
	}

	return nil
}
